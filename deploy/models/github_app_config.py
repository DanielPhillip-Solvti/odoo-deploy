from odoo import models, fields, api
from ..tools import jwt
import time
import requests
from odoo.exceptions import UserError

class GitHubAppConfig(models.Model):
    _name = 'github.app.config'
    _description = 'GitHub App Credentials'

    client_id = fields.Char(string='GitHub Client ID (iss)', help="Starts with Iv1. or found on App Settings")
    installation_id = fields.Char(string='Installation ID')
    private_key = fields.Obscure(string='Private Key (.pem)' )

    def create(self, vals):
        if self.search([], limit=1):
            raise UserError("Only one GitHub App Config record is allowed. Please edit the existing record.")
        return super().create(vals)
    
    def unlink(self):
        raise UserError("Deletion of GitHub App Config is not allowed. Please edit the existing record.")

    def _get_installation_token(self):
        """Generate a short-lived installation token for the GitHub App."""

        private_key = self.private_key
        encoded_jwt = jwt.generate_github_jwt(self.client_id, private_key)

        # 2. Request an installation token from GitHub
        headers = {
            'Authorization': f'Bearer {encoded_jwt}',
            'Accept': 'application/vnd.github+json',
        }
        url = f'https://api.github.com/app/installations/{self.installation_id}/access_tokens'
        response = requests.post(url, headers=headers)

        if response.status_code == 201:
            return response.json().get('token')
        else:
            raise Exception(f"Failed to get installation token: {response.status_code} - {response.text}")