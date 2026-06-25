from datetime import timedelta, timezone

from odoo import fields, models

from ..services.agent_service import ACTIONS

EVENT_AGE_LIMIT_MINUTES = 5


class AgentEvent(models.Model):
    _name = "deploy.event"
    _description = "Agent Event"

    agent_id = fields.Many2one("deploy.agent", required=True)
    action = fields.Selection(selection=ACTIONS, required=True)

    timestamp = fields.Datetime(default=fields.Datetime.now, required=True)
    parameters = fields.Json()
    status = fields.Selection(
        selection=[("pending", "Pending"), ("success", "Success"), ("fail", "Fail")], default="pending"
    )
    message = fields.Text(help="Response from the Agent")

    def create(self, vals):
        self._prune_old_events()
        return super().create(vals)

    def _prune_old_events(self):
        self.search(
            [
                (
                    "timestamp",
                    "<",
                    fields.Datetime.to_string(fields.Datetime.now() - timedelta(minutes=EVENT_AGE_LIMIT_MINUTES)),
                ),
                ("status", "in", ["success", "fail"]),
            ]
        ).unlink()

    def to_dict(self):
        return {
            "id": self.id,
            "action": self.action,
            "timestamp": self.timestamp.replace(tzinfo=timezone.utc).isoformat(),
            "parameters": self.parameters,
        }
