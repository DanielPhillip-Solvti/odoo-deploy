import logging

from odoo import _, api, fields, models
from odoo.exceptions import UserError

from ..services.agent_service import AgentService
from ..services.github_service import GitHubService

_logger = logging.getLogger(__name__)

GITHUB_API = "https://api.github.com"

class Environment(models.Model):
    _name = "deploy.environment"
    _description = "Deployment Environment"

    repository_branch = fields.Char(required=True, help="Git repository branch for the Odoo deployment")
    odoo_version = fields.Selection(
        [
            ("13.0", "13.0"),
            ("14.0", "14.0"),
            ("15.0", "15.0"),
            ("16.0", "16.0"),
            ("17.0", "17.0"),
            ("18.0", "18.0"),
            ("19.0", "19.0"),
            ("20.0", "20.0"),
        ],
        required=True,
        string="Platform Engine Version",
    )
    is_production = fields.Boolean(string="is_production", default=False)
    agent_id = fields.Many2one("deploy.agent", required=True, ondelete="cascade")
    repository_url = fields.Char(related="agent_id.repository_url")
    state = fields.Selection(
        [("deploying", "Deploying"), ("active", "Active"), ("error", "Error")],
        default="deploying",
        readonly=True,
    )
    log_text = fields.Text(string="Deployment Logs", readonly=True)

    _unique_environment = models.Constraint(
        "unique(agent_id, repository_branch)", "Only one environment per project and branch is allowed."
    )

    @api.constrains("is_production", "agent_id")
    def _check_single_production_environment(self):
        for record in self:
            if record.is_production:
                existing_prod_env = self.search(
                    [("agent_id", "=", record.agent_id.id), ("is_production", "=", True), ("id", "!=", record.id)], limit=1
                )
                if existing_prod_env:
                    raise UserError(_("Only one production environment is allowed per project."))

    # Agent Actions ----------------------------------

    def deploy(self):
        return AgentService(self.agent_id).deploy(self.repository_branch, self.is_production)

    def reset_branch(self):
        return AgentService(self.agent_id).reset_branch(self.repository_branch)

    def update_module(self, module_name):
        return AgentService(self.agent_id).update_module(self.repository_branch, module_name)

    def restore_backup(self):
        return AgentService(self.agent_id).restore_backup(self.repository_branch)

    def undeploy(self):
        return AgentService(self.agent_id).undeploy(self.repository_branch)

    def stream_logs(self):
        return AgentService(self.agent_id).stream_logs(self.repository_branch)

    def get_github_commits(self):
        self.ensure_one()
        return GitHubService().get_github_commits(self)
