import secrets
import logging

import requests as http_requests
from werkzeug.utils import redirect as werkzeug_redirect

from odoo import http
from odoo.http import request

_logger = logging.getLogger(__name__)


class GithubOAuthController(http.Controller):

    @http.route('/deploy/github/authorize', type='http', auth='user')
    def github_authorize(self, redirect_url=None, **kwargs):
        ICP = request.env['ir.config_parameter'].sudo()
        client_id = ICP.get_param('deploy.github.client_id')
        if not client_id:
            return request.make_response(
                'GitHub OAuth not configured. Set deploy.github.client_id in system parameters.',
                headers=[('Content-Type', 'text/plain')],
            )

        state = secrets.token_urlsafe(24)
        request.session['github_oauth_state'] = state
        request.session['github_oauth_redirect'] = redirect_url or '/web'

        base_url = ICP.get_param('web.base.url', '').rstrip('/')
        callback_url = f'{base_url}/deploy/github/callback'

        auth_url = (
            'https://github.com/login/oauth/authorize'
            f'?client_id={client_id}'
            f'&redirect_uri={callback_url}'
            f'&scope=repo'
            f'&state={state}'
        )
        return werkzeug_redirect(auth_url)

    @http.route('/deploy/github/callback', type='http', auth='user')
    def github_callback(self, code=None, state=None, **kwargs):
        session_state = request.session.get('github_oauth_state')
        if not state or state != session_state:
            return request.make_response(
                'Invalid OAuth state. Please try connecting again.',
                headers=[('Content-Type', 'text/plain')],
            )

        ICP = request.env['ir.config_parameter'].sudo()
        client_id = ICP.get_param('deploy.github.client_id')
        client_secret = ICP.get_param('deploy.github.client_secret')
        base_url = ICP.get_param('web.base.url', '').rstrip('/')
        callback_url = f'{base_url}/deploy/github/callback'

        try:
            resp = http_requests.post(
                'https://github.com/login/oauth/access_token',
                json={
                    'client_id': client_id,
                    'client_secret': client_secret,
                    'code': code,
                    'redirect_uri': callback_url,
                },
                headers={'Accept': 'application/json'},
                timeout=10,
            )
            data = resp.json()
        except Exception as exc:
            _logger.warning('GitHub token exchange failed: %s', exc)
            return request.make_response(
                'Failed to exchange OAuth code. Please try again.',
                headers=[('Content-Type', 'text/plain')],
            )

        access_token = data.get('access_token')
        if access_token:
            request.env['deploy.github.token'].set_token_for_current_user(access_token)

        redirect_url = request.session.pop('github_oauth_redirect', '/web')
        request.session.pop('github_oauth_state', None)
        return request.redirect(redirect_url)

    @http.route('/deploy/github/disconnect', type='http', auth='user')
    def github_disconnect(self, redirect_url=None, **kwargs):
        request.env['deploy.github.token'].clear_token_for_current_user()
        return request.redirect(redirect_url or '/web')
