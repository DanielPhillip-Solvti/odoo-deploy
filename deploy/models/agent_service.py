_ENDPOINTS = {
    'check_health': {'route': '/health', 'method': 'GET'},
    'reset_repository': {'route': '/reset_repository', 'method': 'POST', 'required_fields': ['branch']},
    'restart_odoo': {'route': '/restart_odoo', 'method': 'POST', 'required_fields': ['branch']},
    'restore_backup': {'route': '/restore_backup', 'method': 'POST', 'required_fields': ['branch']},
    'undeploy': {'route': '/undeploy', 'method': 'POST', 'required_fields': ['branch']},
    'deploy': {'route': '/deploy', 'method': 'POST', 'required_fields': ['branch']},
}

import requests

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

    def _request(self, agent_url, endpoint, return_notification=True, **kwargs):
        endpoint_info = _ENDPOINTS.get(endpoint)

        if not endpoint_info:
            raise ValueError(f"Unknown endpoint: {endpoint}")
        
        if 'required_fields' in endpoint_info:
            missing_fields = [field for field in endpoint_info['required_fields'] if field not in kwargs]
            if missing_fields:
                raise ValueError(f"Missing required fields for {endpoint}: {', '.join(missing_fields)}")
            
        response = getattr(requests, endpoint_info['method'].lower())(
            url=f"{'http://' if not agent_url.startswith('http') else ''}{agent_url}{endpoint_info['route']}",
            json=kwargs
        )

        if response.status_code != 200:
            raise Exception(f"Agent service error: {response.status_code} - {response.text}")
        
        if return_notification:
            return self._notify(response.json()['output'])
        
        return response.json()

    def check_health(self, vm_record):
        return self._request(vm_record.url, 'check_health')
    
    def reset_repository(self, environment_record):
        return self._request(environment_record.url, 'reset_repository', branch=environment_record.repository_branch)

    def restart_odoo(self, environment_record):
        return self._request(environment_record.url, 'restart_odoo', branch=environment_record.repository_branch)

    def restore_backup(self, environment_record):
        return self._request(environment_record.url, 'restore_backup', branch=environment_record.repository_branch)

    def undeploy(self, environment_record):
        return self._request(environment_record.url, 'undeploy', branch=environment_record.repository_branch)

    def deploy(self, environment_record):
        return self._request(environment_record.url, 'deploy', 
            branch=environment_record.repository_branch, 
            addons_repository=environment_record.repository_url, 
            odoo_version=environment_record.odoo_version,
            is_production=environment_record.is_production
        )

    

    