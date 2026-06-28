from odoo import api, fields, models


class WsToken(models.Model):
    _name = "deploy.ws_token"
    _description = "Unified token for WebSocket connections (backup, logs, shell)"

    token = fields.Char(required=True, index=True)
    agent_id = fields.Many2one("deploy.agent", required=True, ondelete="cascade")
    purpose = fields.Selection(
        [("backup", "Backup"), ("logs", "Logs"), ("shell", "Shell")],
        required=True,
    )
    params = fields.Json(default=dict)
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
