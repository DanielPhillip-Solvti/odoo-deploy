from odoo import http, fields
from odoo.http import request
from odoo.modules.module import get_module_path
from ..data_objects.heartbeat import HeartbeatPayload, EventCallbackPayload

import os

import logging
_logger = logging.getLogger(__name__)

class AgentController(http.Controller):
    def _extract_api_key(self):
        auth_header = request.httprequest.headers.get('Authorization')
        if auth_header and auth_header.startswith('Bearer '):
            return auth_header.split(' ', 1)[1]
        return None
    
    @http.route('/agent/heartbeat', type='jsonrpc', auth='public', methods=['POST'], csrf=False)
    def agent_heartbeat(self, **kwargs : HeartbeatPayload):
        token = self._extract_api_key()
        agent = request.env['deploy.agent'].sudo().search([('api_key', '=', token)], limit=1)
        if not agent:
            return {'error': 'Invalid API Key'}

        # heartbeat_payload = HeartbeatPayload(**kwargs)
        agent.sudo().write({
            'last_heartbeat': fields.Datetime.now(),
            # 'heartbeat_payload': heartbeat_payload.dict(),
        })

        # events = agent.get_events(last_event_id=heartbeat_payload.last_event_id)
        return {'status': 'success', 'message': 'Heartbeat received', 'events': []}

    @http.route('/agent/callback/<int:event_id>', type='jsonrpc', auth='public', methods=['POST'], csrf=False)
    def agent_event_callback(self, event_id, **kwargs : EventCallbackPayload ):
        token = self._extract_api_key()
        agent = request.env['deploy.agent'].sudo().search([('api_key', '=', token)], limit=1)
        if not agent:
            return {'error': 'Invalid API Key'}

        event = agent.event_ids.filtered(lambda e: e.id == event_id)
        callback = EventCallbackPayload(**kwargs)

        if not event:
            _logger.warning(f"event repsonse received: {event_id}. status: {callback.status}. message: {callback.message}")

        return {'status': 'success'}

    @http.route('/agent/get_script/<string:script_name>/<string:script_extension>', type='http', auth='public', methods=['GET'], csrf=False)
    def get_script(self, script_name, script_extension):
        """Serve a script from the module directory"""

        filename = f"{script_name}"
        if script_extension != "0":
            filename += f".{script_extension}"

        module_path = get_module_path('deploy')
        script_path = os.path.join(module_path, 'agent_scripts', filename)
        
        if not os.path.exists(script_path):
            return http.Response("Script not found", status=404)
        
        with open(script_path, 'r') as f:
            script_content = f.read()

        return http.Response(
            script_content,
            mimetype='text/plain',
            headers=[
                ('Content-Disposition', f'attachment; filename="{filename}"')
            ]
        )