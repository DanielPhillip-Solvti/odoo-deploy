from odoo import fields, models

from ..services.agent_service import AgentService


class DeployModuleWizard(models.TransientModel):
    _name = "deploy.module.wizard"
    _description = "Install/Update Module Wizard"

    environment_id = fields.Many2one("deploy.environment", required=True)
    module_name = fields.Char(required=True)
    action_type = fields.Selection(
        [("install", "Install"), ("update", "Update")],
        required=True,
        default="install",
    )

    def action_confirm(self):
        self.ensure_one()
        agent = self.environment_id.agent_id
        svc = AgentService(agent)
        branch = self.environment_id.repository_branch
        if self.action_type == "install":
            svc.install_module(branch, self.module_name)
        else:
            svc.update_module(branch, self.module_name)
