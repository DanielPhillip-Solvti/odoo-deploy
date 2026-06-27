import logging
import re

import requests

from ..tools import jwt

_logger = logging.getLogger(__name__)
GITHUB_API = "https://api.github.com"


class GitHubService:
    def __init__(self, env, repository_url):
        self.gh_config = gh_config = env["github.app.config"].search([], limit=1)
        if not gh_config:
            raise ValueError("GitHub App not configured. Please create a github.app.config record.")
        self.repository_url = repository_url
        match = re.search(r"github\.com[/:]([^/]+)/([^/]+?)(?:\.git)?$", repository_url)
        if not match:
            raise ValueError("Invalid GitHub repository URL.")

        self.owner, self.repo = match.group(1), match.group(2)
        self.token = self._get_installation_token(self.gh_config)

    def get_github_commits(self, branch="main"):
        """Fetch commits on the branch compared to its base, with divergence summary.

        :param env: Odoo environment
        :param repository_url: str, the repository URL
        :param branch: str, the branch name
        :return: dict with branch, base_branch, ahead_by, behind_by,
                 merge_base_commit, and commits (branch-only) or an 'error' key
        """

        headers = {
            "Authorization": f"Bearer {self.token}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
        }

        # 4. Fetch repo info for default_branch
        try:
            repo_resp = requests.get(f"{GITHUB_API}/repos/{self.owner}/{self.repo}", headers=headers, timeout=10)
        except requests.RequestException as exc:
            _logger.warning("GitHub repo API request failed: %s", exc)
            return {"error": "request_failed"}

        if repo_resp.status_code in (401, 403):
            return {"error": "bad_credentials_or_forbidden"}
        if repo_resp.status_code != 200:
            return {"error": f"github_api_error_{repo_resp.status_code}"}

        try:
            default_branch = repo_resp.json().get("default_branch", "main")
        except (KeyError, TypeError) as exc:
            _logger.error("Failed to parse repo info: %s", exc)
            return {"error": "unexpected_response_format"}

        # 5. If branch is the default, fetch simple commit list
        if branch == default_branch:
            try:
                resp = requests.get(
                    f"{GITHUB_API}/repos/{self.owner}/{self.repo}/commits",
                    params={"sha": branch, "per_page": 30},
                    headers=headers,
                    timeout=10,
                )
            except requests.RequestException as exc:
                _logger.warning("GitHub API request failed: %s", exc)
                return {"error": "request_failed"}

            if resp.status_code in (401, 403):
                return {"error": "bad_credentials_or_forbidden"}
            if resp.status_code != 200:
                return {"error": f"github_api_error_{resp.status_code}"}

            try:
                commits = [self._parse_commit(c) for c in resp.json()]
            except (KeyError, TypeError) as exc:
                _logger.error("Failed to parse GitHub JSON: %s", exc)
                return {"error": "unexpected_response_format"}

            return {
                "branch": branch,
                "base_branch": default_branch,
                "ahead_by": 0,
                "behind_by": 0,
                "merge_base_commit": None,
                "commits": commits,
            }

        # 6. Compare branch against default branch
        try:
            comp_resp = requests.get(
                f"{GITHUB_API}/repos/{self.owner}/{self.repo}/compare/{default_branch}...{branch}",
                headers=headers,
                timeout=10,
            )
        except requests.RequestException as exc:
            _logger.warning("GitHub compare API request failed: %s", exc)
            return {"error": "request_failed"}

        if comp_resp.status_code in (401, 403):
            return {"error": "bad_credentials_or_forbidden"}
        if comp_resp.status_code != 200:
            return {"error": f"github_api_error_{comp_resp.status_code}"}

        try:
            data = comp_resp.json()
            mb = data.get("merge_base_commit")
            return {
                "branch": branch,
                "base_branch": default_branch,
                "ahead_by": data.get("ahead_by", 0),
                "behind_by": data.get("behind_by", 0),
                "merge_base_commit": self._parse_commit(mb) if mb else None,
                "commits": [self._parse_commit(c) for c in data.get("commits", [])],
            }
        except (KeyError, TypeError) as exc:
            _logger.error("Failed to parse GitHub compare response: %s", exc)
            return {"error": "unexpected_response_format"}

    @staticmethod
    def _parse_commit(c):
        return {
            "sha": c["sha"][:7],
            "sha_full": c["sha"],
            "message": c["commit"]["message"].split("\n")[0],
            "author": c["commit"]["author"]["name"],
            "date": c["commit"]["author"]["date"],
            "url": c["html_url"],
        }

    def list_branches(self):
        """List all branches for a GitHub repository.
        :return: list of branch names
        """
        try:
            response = requests.get(
                f"{GITHUB_API}/repos/{self.owner}/{self.repo}/branches",
                params={"per_page": 100},
                headers={
                    "Authorization": f"Bearer {self.token}",
                    "Accept": "application/vnd.github+json",
                    "X-GitHub-Api-Version": "2022-11-28",
                },
                timeout=10,
            )
        except requests.RequestException as exc:
            _logger.warning("Failed to list branches: %s", exc)
            return []

        if response.status_code != 200:
            _logger.warning("GitHub API error listing branches: %s", response.status_code)
            return []

        return [b["name"] for b in response.json()]

    def _get_installation_token(self, gh_config):
        private_key = gh_config.private_key
        encoded_jwt = jwt.generate_github_jwt(gh_config.client_id, private_key)

        headers = {
            "Authorization": f"Bearer {encoded_jwt}",
            "Accept": "application/vnd.github+json",
        }
        url = f"https://api.github.com/app/installations/{gh_config.installation_id}/access_tokens"
        response = requests.post(url, headers=headers)

        if response.status_code == 201:
            return response.json().get("token")
        else:
            raise Exception(f"Failed to get installation token: {response.status_code} - {response.text}")
