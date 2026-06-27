from odoo import _, fields, models
from odoo.exceptions import UserError


class GitHubAppConfig(models.Model):
    _name = "github.app.config"
    _description = "GitHub App Credentials"

    client_id = fields.Char(string="GitHub Client ID (iss)", help="Starts with Iv1. or found on App Settings")
    installation_id = fields.Char(string="Installation ID")
    private_key = fields.Obscure(string="Private Key (.pem)")

    def create(self, vals):
        if self.search([], limit=1):
            raise UserError(_("Only one GitHub App Config record is allowed. Please edit the existing record."))
        return super().create(vals)

    def unlink(self):
        raise UserError(_("Deletion of GitHub App Config is not allowed. Please edit the existing record."))
