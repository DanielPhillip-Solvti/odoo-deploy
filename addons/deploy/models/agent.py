import json
import logging
import secrets

from odoo import _, api, fields, models
from odoo.exceptions import UserError

from ..services.agent_service import AgentService
from ..services.github_service import GitHubService

_logger = logging.getLogger(__name__)


class Agent(models.Model):
    _name = "deploy.agent"
    _description = "Client Project"

    name = fields.Char(required=True)
    api_key = fields.Obscure(readonly=True)
    bootstrap_script = fields.Text(compute="_compute_bootstrap_script", readonly=True)
    repository_url = fields.Char(
        readonly=True,
        help="Git repository URL. Set by the agent during bootstrap.",
    )
    ws_url = fields.Char(
        help="Agent WebSocket base URL. Example: wss://example.com or ws://localhost:9876",
    )

    last_heartbeat = fields.Datetime()
    heartbeat_payload = fields.Json()

    status = fields.Selection(
        selection=[("offline", "Offline"), ("active", "Active"), ("error", "Error")],
        compute="_compute_status",
        default="offline",
    )

    event_ids = fields.One2many("deploy.event", "agent_id")

    def _compute_status(self):
        for record in self:
            if not record.last_heartbeat:
                record.status = "offline"
            else:
                time_since_last_heartbeat = fields.Datetime.now() - record.last_heartbeat
                if time_since_last_heartbeat.total_seconds() > 300:
                    record.status = "offline"
                else:
                    record.status = "active"

    def get_events(self, last_event_id=None):
        self.ensure_one()
        events = self.event_ids.filtered(lambda e: e.status == "pending")
        if last_event_id is not None:
            events = events.filtered(lambda e: e.id > last_event_id)
        return [event.to_dict() for event in events.sorted(key=lambda e: e.timestamp)]

    def generate_api_key(self):
        self.ensure_one()
        self.api_key = secrets.token_urlsafe(32)

    def _compute_bootstrap_script(self):
        for record in self:
            if not record.api_key:
                record.bootstrap_script = (
                    "Generate an authentication key below and save to create the bootstrap script."
                )
                continue

            base_url = self.env["ir.config_parameter"].sudo().get_param("web.base.url") or ""

            if not base_url:
                record.bootstrap_script = (
                    "Configure the web.base.url system parameter to generate the bootstrap script."
                )
                continue

            base_url = base_url.rstrip("/")

            record.bootstrap_script = (
                f'curl -fsSL "{base_url}/agent/get_script/bootstrap/sh" -o setup.sh && '
                f"chmod +x setup.sh && "
                f"./setup.sh "
                f'--odoo-url "{base_url}" '
                f'--api-key "{record.api_key}"'
            )

    @api.model_create_multi
    def create(self, vals_list):
        result = super().create(vals_list)
        for record in result:
            record.generate_api_key()
        return result

    def _deployed_branches_from_heartbeat(self):
        deployed = set()
        payload = self._parse_heartbeat()
        pb = payload.get("production_branch") or {}
        if pb.get("branch"):
            deployed.add(pb["branch"])
        for sb in payload.get("staging_branches") or []:
            if sb.get("branch"):
                deployed.add(sb["branch"])
        return deployed

    def action_open_dashboard(self):
        self.ensure_one()
        return {
            "type": "ir.actions.client",
            "tag": "deploy.dashboard",
            "params": {"agent_id": self.id},
        }

    def get_undeployed_branches(self):
        self.ensure_one()
        try:
            all_branches = GitHubService(self.env, self.repository_url).list_branches()
        except Exception as e:
            _logger.warning("Failed to get undeployed branches: %s", e)
            return []
        deployed = self._deployed_branches_from_heartbeat()
        return [b for b in all_branches if b not in deployed]

    def deploy_branch(self, branch, is_production=False):
        self.ensure_one()
        event = AgentService(self).deploy(branch, is_production)
        return {"event_id": event.id}

    def undeploy_branch(self, branch):
        self.ensure_one()
        event = AgentService(self).undeploy(branch)
        return {"event_id": event.id}

    def get_github_commits(self, branch):
        self.ensure_one()
        return GitHubService(self.env, self.repository_url).get_github_commits(branch)

    def get_git_commands(self, branch):
        self.ensure_one()
        repo_url = self.repository_url or ""
        branch = branch or ""
        return [
            {"key": "clone", "label": "Clone", "icon": "fa-copy", "command": f"git clone {repo_url}"},
            {
                "key": "fork",
                "label": "Fork",
                "icon": "fa-code-fork",
                "command": f"gh repo fork {repo_url} --clone=false",
            },
            {
                "key": "merge",
                "label": "Merge",
                "icon": "fa-code-merge",
                "command": f"git checkout {branch} && git pull origin {branch}",
            },
            {"key": "ssh", "label": "SSH", "icon": "fa-terminal", "command": f"ssh -t deploy@{branch}.local"},
            {
                "key": "sql",
                "label": "SQL",
                "icon": "fa-database",
                "command": f"docker exec -it $(docker ps -q -f name={branch}) psql -U odoo -d {branch}",
            },
            {
                "key": "submodule",
                "label": "Submodule",
                "icon": "fa-puzzle-piece",
                "command": "git submodule update --init --recursive",
            },
            {"key": "delete", "label": "Delete", "icon": "fa-trash", "command": f"odoosh deploy destroy {branch}"},
        ]

    def _broadcast_heartbeat_via_bus(self):
        self.ensure_one()
        payload = self._parse_heartbeat()
        if not payload:
            return
        pb = payload.get("production_branch") or {}
        sbs = payload.get("staging_branches") or []
        envs = []
        if pb.get("branch"):
            envs.append(
                {
                    "branch": pb["branch"],
                    "odoo_version": pb.get("odoo_version", ""),
                    "status": pb.get("status", ""),
                    "is_production": True,
                }
            )
        for sb in sbs:
            if sb.get("branch"):
                envs.append(
                    {
                        "branch": sb["branch"],
                        "odoo_version": sb.get("odoo_version", ""),
                        "status": sb.get("status", ""),
                        "is_production": False,
                    }
                )
        self.env["bus.bus"]._sendone(
            f"deploy_agent_{self.id}",
            "deploy.heartbeat",
            {
                "agent_id": self.id,
                "agent_name": self.name,
                "status": self.status,
                "last_heartbeat": str(self.last_heartbeat) if self.last_heartbeat else None,
                "environments": envs,
                "backups": payload.get("backups", []),
            },
        )

    # Agent Actions ----------------------------------
    def backup(self, with_dump=False, branch=None):
        if with_dump:
            payload = self._parse_heartbeat()
            prod_branch = (payload or {}).get("production_branch", {}).get("branch")
            if not branch or branch != prod_branch:
                raise UserError(_("Dump backup is only allowed on the production branch"))
        return AgentService(self).backup(with_dump, branch)

    def _parse_heartbeat(self):
        payload = self.heartbeat_payload
        if isinstance(payload, str):
            try:
                payload = json.loads(payload)
            except (json.JSONDecodeError, TypeError):
                return {}
        return payload if isinstance(payload, dict) else {}

    def request_ws_token(self, purpose, params=None):
        self.ensure_one()

        token = secrets.token_hex(32)
        expiry = fields.Datetime.add(fields.Datetime.now(), seconds=60)
        base = (self.ws_url or "").rstrip("/") or "ws://localhost:9876"
        ws_url = f"{base}/{purpose}-ws"

        self.env["deploy.ws_token"].create(
            {
                "token": token,
                "agent_id": self.id,
                "purpose": purpose,
                "params": params or {},
                "expiry": expiry,
            }
        )
        return {"token": token, "ws_url": ws_url}

    # Branch Actions ----------------------------------
    def deploy(self, branch, is_production=False):
        return AgentService(self).deploy(branch, is_production)

    def undeploy(self, branch):
        return AgentService(self).undeploy(branch)

    def restore_backup(self, branch):
        return AgentService(self).restore_backup(branch)

    def reset_branch(self, branch):
        return AgentService(self).reset_branch(branch)

    def update_module(self, branch, module_name):
        return AgentService(self).update_module(branch, module_name)

    # buttons
    def backup_no_dump(self):
        return self.backup(with_dump=False)

    def backup_with_dump(self):
        payload = self._parse_heartbeat()
        branch = payload.get("production_branch", {}).get("branch") if payload else None
        return self.backup(with_dump=True, branch=branch)
