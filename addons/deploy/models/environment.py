from odoo import fields, models

from ..services.agent_service import AgentService


class DeployEnvironment(models.Model):
    _name = "deploy.environment"
    _description = "Deployment Environment"
    _rec_name = "display_name"
    _order = "is_production DESC, repository_branch"

    agent_id = fields.Many2one("deploy.agent", required=True, ondelete="cascade")
    repository_branch = fields.Char(required=True)
    odoo_version = fields.Char()
    is_production = fields.Boolean()
    state = fields.Char()
    stale = fields.Boolean(default=False)
    display_name = fields.Char(compute="_compute_display_name")
    environment_type = fields.Char(compute="_compute_environment_type")
    event_ids = fields.One2many("deploy.event", "environment_id")

    def _compute_display_name(self):
        for record in self:
            parts = []
            if record.agent_id:
                parts.append(record.agent_id.display_name or record.agent_id.name)
            if record.repository_branch:
                parts.append(record.repository_branch)
            record.display_name = " / ".join(parts)

    def _compute_environment_type(self):
        for record in self:
            record.environment_type = "Production" if record.is_production else "Staging"

    def action_open_agent(self):
        self.ensure_one()
        return {
            "type": "ir.actions.act_window",
            "res_model": "deploy.agent",
            "res_id": self.agent_id.id,
            "view_mode": "form",
        }

    def action_undeploy(self):
        self.ensure_one()
        agent = self.agent_id
        event = AgentService(agent).undeploy(self.repository_branch)
        return {"event_id": event.id}

    def action_reset_branch(self):
        self.ensure_one()
        agent = self.agent_id
        event = AgentService(agent).reset_branch(self.repository_branch)
        return {"event_id": event.id}

    def action_update_module(self):
        self.ensure_one()
        agent = self.agent_id
        ctx = self.env.context
        module_name = ctx.get("default_module_name", "")
        if not module_name:
            return {
                "type": "ir.actions.act_window",
                "res_model": "deploy.environment.update.module.wizard",
                "view_mode": "form",
                "target": "new",
                "context": {
                    "default_environment_id": self.id,
                    "default_branch": self.repository_branch,
                },
            }
        event = AgentService(agent).update_module(self.repository_branch, module_name)
        return {"event_id": event.id}

    def action_install_module(self):
        self.ensure_one()
        agent = self.agent_id
        ctx = self.env.context
        module_name = ctx.get("default_module_name", "")
        if not module_name:
            return {
                "type": "ir.actions.act_window",
                "res_model": "deploy.environment.install.module.wizard",
                "view_mode": "form",
                "target": "new",
                "context": {
                    "default_environment_id": self.id,
                    "default_branch": self.repository_branch,
                },
            }
        event = AgentService(agent).install_module(self.repository_branch, module_name)
        return {"event_id": event.id}

    def action_restore_backup(self):
        self.ensure_one()
        agent = self.agent_id
        event = agent.restore_backup(self.repository_branch)
        return {"event_id": event.id}
