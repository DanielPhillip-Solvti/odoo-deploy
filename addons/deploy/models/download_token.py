from odoo import api, fields, models


class DownloadToken(models.Model):
    _name = "deploy.download_token"
    _description = "Temporary token for backup file download"

    token = fields.Char(required=True, index=True)
    agent_id = fields.Many2one("deploy.agent", required=True, ondelete="cascade")
    filename = fields.Char(required=True)
    expiry = fields.Datetime(required=True)
    used = fields.Boolean(default=False)

    def is_valid(self):
        self.ensure_one()
        return not self.used and self.expiry > fields.Datetime.now()

    def mark_used(self):
        self.ensure_one()
        self.used = True

    @api.model
    def _prune_expired(self):
        cutoff = fields.Datetime.subtract(hours=1)
        expired = self.search([("expiry", "<", cutoff)])
        expired.unlink()
