import requests

_ENDPOINTS = {
    'check_health': {'route': '/health', 'method': 'GET'},
    'backup': {'route': '/backup', 'method': 'POST', 'required_fields': ['branch','with_dump']},
    'deploy': {'route': '/deploy', 'method': 'POST', 'required_fields': ['branch', 'github_token']},
    'download_dump': {'route': '/dump', 'method': 'GET', 'required_fields': ['is_production']},
    'reset_branch': {'route': '/reset_branch', 'method': 'POST', 'required_fields': ['branch', 'github_token']},
    'restore_backup': {'route': '/restore_backup', 'method': 'POST', 'required_fields': ['branch', 'is_production']},
    'stream_logs': {'route': '/logs', 'method': 'GET', 'required_fields': ['branch']},
    'undeploy': {'route': '/undeploy', 'method': 'POST', 'required_fields': ['branch']},
    'update_module': {'route': '/update_module', 'method': 'POST', 'required_fields': ['branch']},
}

class AgentService:
    def _notify(self, message):
        return {
            'type': 'ir.actions.client',
            'tag': 'display_notification',
            'params': {
                'message': message,
                'type': 'info',
            },
        }

    def _get_active_token(self, env):
        """Helper to fetch a fresh GitHub installation token safely."""
        gh_config = env['github.app.config'].search([], limit=1)
        if not gh_config:
            raise ValueError("Action failed: GitHub App Configuration is missing in settings.")
        return gh_config._get_installation_token()

    def _request(self, agent_url, endpoint, return_notification=True, **kwargs):
        endpoint_info = _ENDPOINTS.get(endpoint)

        if not endpoint_info:
            raise ValueError(f"Unknown endpoint: {endpoint}")
        
        if 'required_fields' in endpoint_info:
            missing_fields = [field for field in endpoint_info['required_fields'] if field not in kwargs]
            if missing_fields:
                raise ValueError(f"Missing required fields for {endpoint}: {', '.join(missing_fields)}")
            
        # Clean up URL protocol handling
        base_url = f"{'http://' if not agent_url.startswith('http') else ''}{agent_url}"
        full_url = f"{base_url}{endpoint_info['route']}"
        method = endpoint_info['method'].lower()

        # Handle payload formatting depending on GET vs POST
        request_args = {'timeout': 30}
        if method == 'get':
            request_args['params'] = kwargs
        else:
            request_args['json'] = kwargs

        response = getattr(requests, method)(url=full_url, **request_args)

        if response.status_code != 200:
            raise Exception(f"Agent service error: {response.status_code} - {response.text}")
        
        if return_notification:
            return self._notify(response.json().get('output', 'Action completed successfully.'))
        
        return response.json()

    # -------------------------------------------------------------------------
    # Operational Service Methods
    # -------------------------------------------------------------------------

    def check_health(self, vm_record):
        return self._request(vm_record.url, 'check_health')
    
    def backup(self, vm_record, with_dump=False):
        return self._request(vm_record.url, 'backup', with_dump=with_dump)

    def deploy(self, environment_record):
        token = self._get_active_token(environment_record.env)
        return self._request(
            environment_record.url, 'deploy', 
            branch=environment_record.repository_branch, 
            addons_repository=environment_record.repository_url, 
            odoo_version=environment_record.odoo_version,
            is_production=environment_record.is_production,
            github_token=token
        )

    def download_dump(self, vm_record, production):
        return self._request(vm_record.url, 'download_dump', is_production=production, return_notification=False)

    def reset_branch(self, environment_record):
        token = self._get_active_token(environment_record.env)
        return self._request(
            environment_record.url, 'reset_branch', 
            branch=environment_record.repository_branch,
            github_token=token
        )

    def restore_backup(self, environment_record):
        return self._request(environment_record.url, 'restore_backup', branch=environment_record.repository_branch, is_production=environment_record.is_production)

    def stream_logs(self, environment_record):
        return self._request(environment_record.url, 'stream_logs', branch=environment_record.repository_branch, return_notification=False)

    def undeploy(self, environment_record):
        return self._request(environment_record.url, 'undeploy', branch=environment_record.repository_branch)

    def update_module(self, environment_record, module_to_update='all'):
        return self._request(
            environment_record.url, 'update_module', 
            branch=environment_record.repository_branch,
            module=module_to_update
        )