from odoo import api, fields, models

from ..services.agent_service import AgentService


class Agent(models.Model):
    _name = "deploy.agent"
    _description = "Client Project"

    name = fields.Char(required=True)
    api_key = fields.Obscure(readonly=True)
    production_environment_id = fields.Many2one("deploy.environment", readonly=True)
    staging_environment_ids = fields.One2many("deploy.environment", "agent_id", domain=[("is_production", "=", False)])
    bootstrap_script = fields.Text(compute="_compute_bootstrap_script", readonly=True)
    repository_url = fields.Char(
        required=True,
        help="Git repository URL for the Odoo deployment including branch or tag if necessary",
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
                if time_since_last_heartbeat.total_seconds() > 300:  # 5 minutes threshold
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
        self.api_key = "generated_api_key" + str(self.id)

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

    # Agent Actions ----------------------------------
    def check_health(self):
        return AgentService(self).check_health(self)

    def backup(self, with_dump=False):
        return AgentService(self).backup(self, with_dump)

    def download_dump(self, production):
        return AgentService(self).download_dump(self, production=production)

    # buttons
    def backup_no_dump(self):
        return self.backup(with_dump=False)

    def backup_with_dump(self):
        return self.backup(with_dump=True)

    def download_production_dump(self):
        return self.download_dump(production=True)

    def download_neutralised_dump(self):
        return self.download_dump(production=False)
