import re
import requests
import logging

from odoo import models, fields, api
from .agent_service import AgentService

_logger = logging.getLogger(__name__)

GITHUB_API = 'https://api.github.com'

class Environment(models.Model):
    _name = 'deploy.environment'
    _description = 'Deployment Environment'

    url = fields.Char(string='Agent URL', related='vm_id.url', readonly=True)
    repository_branch = fields.Char(required=True, help='Git repository branch for the Odoo deployment')
    odoo_version = fields.Char(required=True)
    is_production = fields.Boolean( default=False)
    vm_id = fields.Many2one('deploy.vm', required=True, ondelete='cascade')
    repository_url = fields.Char(related='vm_id.repository_url')

    def reset_repository(self):
        return AgentService().reset_repository(self)

    def restart_odoo(self):
        return AgentService().restart_odoo(self)

    def restore_backup(self):
        return AgentService().restore_backup(self)

    def undeploy(self):
        return AgentService().undeploy(self)

    def deploy(self):
        return AgentService().deploy(self)

    def get_github_commits(self):
        """Fetch recent commits for this environment's repository branch via GitHub App Installation Token."""
        self.ensure_one()
        
        # 1. Fetch the GitHub App Configuration
        gh_config = self.env['github.app.config'].search([], limit=1)
        if not gh_config:
            return {'error': 'github_app_not_configured'}

        # 2. Extract owner and repo using your regex
        repo_url = self.repository_url or ''
        match = re.search(r'github\.com[/:]([^/]+)/([^/]+?)(?:\.git)?$', repo_url)
        if not match:
            return {'error': 'invalid_repository_url'}

        owner, repo = match.group(1), match.group(2)
        branch = self.repository_branch or 'main'

        # 3. Request a short-lived system token using your GitHub App credentials
        try:
            token = gh_config._get_installation_token()
        except Exception as exc:
            _logger.error('Failed to generate GitHub App installation token: %s', exc)
            return {'error': 'token_generation_failed'}

        # 4. Make the request to GitHub
        try:
            response = requests.get(
                f'{GITHUB_API}/repos/{owner}/{repo}/commits',
                params={'sha': branch, 'per_page': 30},
                headers={
                    'Authorization': f'Bearer {token}',
                    'Accept': 'application/vnd.github+json',
                    'X-GitHub-Api-Version': '2022-11-28',
                },
                timeout=10,
            )
        except requests.RequestException as exc:
            _logger.warning('GitHub API request failed: %s', exc)
            return {'error': 'request_failed'}

        # 5. Handle authentication and API errors
        if response.status_code in (401, 403):
            _logger.error('GitHub App authentication failed (401/403). Check App ID, Installation ID, or Key.')
            return {'error': 'bad_credentials_or_forbidden'}

        if response.status_code != 200:
            return {'error': f'github_api_error_{response.status_code}'}

        # 6. Parse and return commits cleanly
        try:
            commits = [
                {
                    'sha': c['sha'][:7],
                    'sha_full': c['sha'],
                    'message': c['commit']['message'].split('\n')[0],
                    'author': c['commit']['author']['name'],
                    'date': c['commit']['author']['date'],
                    'url': c['html_url'],
                }
                for c in response.json()
            ]
            return {'commits': commits}
        except (KeyError, TypeError) as exc:
            _logger.error('Failed to parse GitHub JSON payload response: %s', exc)
            return {'error': 'unexpected_response_format'}
