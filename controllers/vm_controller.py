from odoo import http, _
from odoo.exceptions import UserError
from odoo.http import request, Response

class VMController(http.Controller):
    @http.route('/register_agent', type='json', auth='public')
    def register_agent(self, **kwargs):
        vm_id = kwargs.get('vm_id', False)
        otp = kwargs.get('otp', False)
        ip_address = kwargs.get('ip_address', False)
        if not (vm_id and otp and ip_address):
            raise UserError(_("vm_id, otp and ip address must be provided to allow an agent registration"))
        vm = request.env["deploy.vm"].browse(vm_id)
        if not vm:
            raise UserError(_("vm with id %{id}s not found", id=vm_id))
        return vm.sudo().exchange_otp(otp, ip_address)

    @http.route('/agent/install/<string:otp>', type='http', auth='public', methods=['GET'])
    def install_agent(self, otp, **kwargs):
        vm = request.env["deploy.vm"].sudo().search([('otp', '=', otp)], limit=1)
        if not vm:
            return Response("Invalid or expired install token", status=404)

        odoo_url = request.httprequest.host_url.rstrip('/')
        script = f"""#!/bin/bash
set -e

INSTALL_DIR=/opt/agent
AGENT_URL="{odoo_url}/agent/binary"

echo "Installing Staccato agent..."

# Install docker if not present
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com | sh
fi

mkdir -p "$INSTALL_DIR"

# Download agent binary
curl -fsSL "$AGENT_URL" -o "$INSTALL_DIR/agent"
chmod +x "$INSTALL_DIR/agent"

# Write env file
cat > "$INSTALL_DIR/vm.env" <<EOF
ODOO_URL={odoo_url}
AGENT_VM_ID={vm.id}
AGENT_OTP={otp}
EOF

# Install systemd service
cat > /etc/systemd/system/staccato-agent.service <<EOF
[Unit]
Description=Staccato Agent
After=network.target docker.service

[Service]
EnvironmentFile=$INSTALL_DIR/vm.env
ExecStart=$INSTALL_DIR/agent
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable staccato-agent
systemctl start staccato-agent

echo "Agent installed and started."
"""
        return Response(script, content_type='text/plain')
