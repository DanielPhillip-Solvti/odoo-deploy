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

    url = fields.Char(string="Agent URL", related="vm_id.url", readonly=True)
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
    vm_id = fields.Many2one("deploy.vm", required=True, ondelete="cascade")
    repository_url = fields.Char(related="vm_id.repository_url")
    state = fields.Selection(
        [("draft", "Draft"), ("deploying", "Deploying"), ("active", "Active"), ("error", "Error")],
        default="draft",
        readonly=True,
    )
    log_text = fields.Text(string="Deployment Logs", readonly=True)

    _unique_environment = models.Constraint(
        "unique(vm_id, repository_branch)", "Only one environment per project and branch is allowed."
    )

    @api.constrains("is_production", "vm_id")
    def _check_single_production_environment(self):
        for record in self:
            if record.is_production:
                existing_prod_env = self.search(
                    [("vm_id", "=", record.vm_id.id), ("is_production", "=", True), ("id", "!=", record.id)], limit=1
                )
                if existing_prod_env:
                    raise UserError(_("Only one production environment is allowed per project."))

    # Agent Actions ----------------------------------

    def check_health(self):
        return AgentService().check_env_health(self)

    def deploy(self):
        return AgentService().deploy(self)

    def reset_branch(self):
        return AgentService().reset_branch(self)

    def restart_odoo(self):
        return AgentService().restart_odoo(self)

    def restore_backup(self):
        return AgentService().restore_backup(self)

    def undeploy(self):
        return AgentService().undeploy(self)

    def stream_logs(self):
        return AgentService().stream_logs(self)

    def get_github_commits(self):
        self.ensure_one()
        return GitHubService().get_github_commits(self)
