from odoo import http
from odoo.http import request, Response
from odoo.exceptions import UserError
from odoo.tools.translate import _

class VMController(http.Controller):

    @http.route('/register_agent', type='jsonrpc', auth='public')
    def register_agent(self, **kwargs):
        vm_id = kwargs.get('vm_id', False)
        otp = kwargs.get('otp', False)
        ip_address = kwargs.get('ip_address', False)
        
        if not (vm_id and otp and ip_address):
            raise UserError(_("vm_id, otp and ip address must be provided to allow an agent registration"))
            
        vm = request.env["deploy.vm"].browse(vm_id)
        if not vm:
            raise UserError(_("vm with id %s not found", vm_id))
            
        return vm.sudo().exchange_otp(otp, ip_address)

    @http.route('/agent/install/<string:otp>', type='http', auth='public', methods=['GET'])
    def install_agent(self, otp, **kwargs):
        """Generates a contextual bash bootstrap payload to orchestrate Agent, Postgres, and Caddy."""
        vm = request.env["deploy.vm"].sudo().search([('otp', '=', otp)], limit=1)
        if not vm:
            return Response("Invalid or expired install token", status=404)

        base_url = request.env['ir.config_parameter'].sudo().get_param('web.base.url')
        github_token = request.env['github.app.config'].sudo().search([], limit=1)._get_installation_token()
        
        bootstrap_script = f"""#!/usr/bin/env bash
set -e

echo "⚙️ Checking server runtime pre-requisites..."
if ! command -v docker &> /dev/null; then
    echo "❌ Error: Docker engine is missing. Install Docker before continuing." >&2
    exit 1
fi

if ! command -v docker compose &> /dev/null; then
    echo "❌ Error: Docker Compose plugin is missing." >&2
    exit 1
fi

# --- 1. Workspace Configuration & Permissions ---
TARGET_DIR="/data/deploy-agent"
echo "📂 Creating application environment workspace directory inside $TARGET_DIR..."
sudo mkdir -p "$TARGET_DIR"
sudo mkdir -p /var/run/postgresql
sudo chmod 777 /var/run/postgresql  # Ensures Postgres container can write local unix sockets
sudo chown -R $USER:$USER "$TARGET_DIR"
cd "$TARGET_DIR"

# --- 2. Write Environment Configuration ---
echo "📝 Writing structural .env variables payload..."
cat << 'EOF' > .env
# Agent Config
ODOO_URL={base_url}
AGENT_VM_ID={vm.id}
AGENT_OTP={otp}

# Database Config
POSTGRES_USER=odoo
POSTGRES_PASSWORD=odoo
EOF

# --- 3. Write Initial Base Caddyfile ---
echo "📝 Initializing standard Caddyfile route template..."
cat << 'EOF' > Caddyfile
# Global configuration block
{{
    email admin@example.com
}}

# Basic placeholder catch-all route over HTTP
:80 {{
    respond "Infrastructure initialized. Deploy environments via Odoo to map domains." 200
}}
EOF

# --- 4. Write Consolidated Multi-Service Docker Compose ---
echo "📝 Creating multi-tier infrastructure layer (docker-compose.yml)..."
cat << 'EOF' > docker-compose.yml
services:
  agent:
    image: ghcr.io/danielphillip-solvti/deploy-agent:main
    container_name: deploy-agent
    restart: unless-stopped
    networks:
      - web
      - deploy-db-network
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /code:/code
    env_file:
      - .env

  db:
    image: pgvector/pgvector:pg18-trixie
    container_name: deploy-db
    environment:
      - POSTGRES_USER=${{POSTGRES_USER}}
      - POSTGRES_PASSWORD=${{POSTGRES_PASSWORD}}
  volumes:
      - deploy-db:/var/lib/postgresql/data:z
      - /var/run/postgresql:/var/run/postgresql
    restart: unless-stopped
    networks:
      - deploy-db-network
    env_file:
      - .env

  caddy:
    image: caddy:latest
    container_name: deploy-caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy_data:/data
      - caddy_config:/config
    networks:
      - web

volumes:
  caddy_data:
  caddy_config:
  deploy-db:

networks:
  web:
    name: web
    driver: bridge
  deploy-db-network:
    name: deploy-db-network
    driver: bridge
EOF

# --- login to GHCR if token is present (optional, but speeds up first pull) ---
echo "{github_token}" | docker login ghcr.io -u x-access-token --password-stdin

# --- 5. Pull Images and Launch ---
echo "🚀 Orchestrating backend service cluster deployment via docker compose..."
docker compose pull
docker compose up -d

echo "✅ Success! Core services (Agent, DB, Reverse Proxy) are active."
echo "📋 View runtime matrix status with: 'cd $TARGET_DIR && docker compose ps'"
"""

        return Response(
            bootstrap_script.strip(),
            headers=[
                ('Content-Type', 'text/plain; charset=utf-8'),
                ('Content-Disposition', 'inline; filename="install_agent.sh"')
            ]
        )