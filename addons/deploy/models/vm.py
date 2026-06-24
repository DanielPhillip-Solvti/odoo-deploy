import random
import string

from odoo import api, fields, models

from ..services.agent_service import AgentService


class VM(models.Model):
    _name = "deploy.vm"
    _description = "Client Project"

    name = fields.Char(required=True)
    url = fields.Char(string="URL", readonly=False)
    environment_ids = fields.One2many("deploy.environment", "vm_id", string="Environments")
    otp = fields.Char(string="One-Time Password", readonly=True)
    bootstrap_script = fields.Text(compute="_compute_bootstrap_script", readonly=True)
    capabilities = fields.Json(help="Commands and functions that the server can execute")
    status = fields.Selection(
        [("offline", "Offline"), ("pending", "Pending"), ("active", "Active"), ("error", "Error")],
        default="pending",
        readonly=True,
    )
    repository_url = fields.Char(
        string="Repository URL",
        required=True,
        help="Git repository URL for the Odoo deployment including branch or tag if necessary",
    )

    def _compute_bootstrap_script(self):
        for record in self:
            if not record.otp:
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

            curl_cmd = (
                f'curl -fsSL "{base_url}/agent/get_script/bootstrap/sh" -o setup.sh && '
                f"chmod +x setup.sh && "
                f"./setup.sh "
                f'--odoo-url "{base_url}" '
                f'--vm-id "{record.id}" '
                f'--otp "{record.otp}" '
                f'--env "production" '
                f'--postgres-user "postgres" '
                f'--postgres-password "odoo" '
            )

            record.bootstrap_script = curl_cmd

    def generate_otp(self):
        self.otp = self._generate_otp()

    def _generate_otp(self):
        return "".join(random.choices(string.ascii_uppercase + string.digits, k=6))

    def exchange_otp(self, provided_otp, url):
        self.ensure_one()
        if self.otp == provided_otp:
            self.otp = False
            self.url = url
            self.status = "active"
            return True
        return False

    @api.model_create_multi
    def create(self, vals_list):
        for vals in vals_list:
            vals["otp"] = self._generate_otp()
        return super().create(vals_list)

    # Agent Actions ----------------------------------

    def check_health(self):
        return AgentService().check_health(self)

    def backup(self, with_dump=False):
        return AgentService().backup(self, with_dump=with_dump)

    def download_dump(self, production):
        return AgentService().download_dump(self, production=production)

    # buttons
    def backup_no_dump(self):
        return self.backup(with_dump=False)

    def backup_with_dump(self):
        return self.backup(with_dump=True)

    def download_production_dump(self):
        return self.download_dump(production=True)

    def download_neutralised_dump(self):
        return self.download_dump(production=False)
