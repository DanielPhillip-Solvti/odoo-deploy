import logging
import re
import requests
from ..tools import jwt

_logger = logging.getLogger(__name__)
GITHUB_API = "https://api.github.com"

class GitHubService:
    def get_github_commits(self, environment_record):
        """Fetch recent commits for a given environment record's repository branch.
        
        :param environment_record: Browse record of 'deploy.environment'
        :return: dict containing either a list of 'commits' or an 'error' key
        """
        # Ensure we have a valid single record context passed in
        environment_record.ensure_one()
        env = environment_record.env
        
        # 1. Fetch the GitHub App Configuration globally
        gh_config = env['github.app.config'].search([], limit=1)
        if not gh_config:
            _logger.warning('GitHub commit fetch skipped: github.app.config record not found.')
            return {'error': 'github_app_not_configured'}

        # 2. Extract owner and repo using regex from the record's URL
        repo_url = environment_record.repository_url or ''
        match = re.search(r'github\.com[/:]([^/]+)/([^/]+?)(?:\.git)?$', repo_url)
        if not match:
            return {'error': 'invalid_repository_url'}

        owner, repo = match.group(1), match.group(2)
        branch = environment_record.repository_branch or 'main'

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
            _logger.error('GitHub App authentication failed (401/403). Check Client ID, Installation ID, or Key.')
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

    def get_installation_token(self, gh_config):
        private_key = gh_config.private_key
        encoded_jwt = jwt.generate_github_jwt(gh_config.client_id, private_key)

        headers = {
            'Authorization': f'Bearer {encoded_jwt}',
            'Accept': 'application/vnd.github+json',
        }
        url = f'https://api.github.com/app/installations/{gh_config.installation_id}/access_tokens'
        response = requests.post(url, headers=headers)

        if response.status_code == 201:
            return response.json().get('token')
        else:
            raise Exception(f"Failed to get installation token: {response.status_code} - {response.text}")