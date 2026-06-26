import logging
import os

from odoo import fields, http
from odoo.http import request
from odoo.modules.module import get_module_path

from ..data_objects.heartbeat import EventCallbackPayload, HeartbeatPayload

_logger = logging.getLogger(__name__)


class AgentController(http.Controller):
    def _extract_api_key(self):
        auth_header = request.httprequest.headers.get("Authorization")
        if auth_header and auth_header.startswith("Bearer "):
            return auth_header.split(" ", 1)[1]
        return None

    @http.route("/agent/heartbeat", type="jsonrpc", auth="public", methods=["POST"], csrf=False)
    def agent_heartbeat(self, **kwargs: HeartbeatPayload):
        token = self._extract_api_key()
        agent = request.env["deploy.agent"].sudo().search([("api_key", "=", token)], limit=1)
        if not agent:
            return {"error": "Invalid API Key"}

        heartbeat_payload = HeartbeatPayload(**kwargs)
        self._apply_heartbeat(agent.sudo(), heartbeat_payload)

        events = agent.get_events(last_event_id=heartbeat_payload.last_event_id)
        return {"status": "success", "message": "Heartbeat received", "events": events}

    @http.route("/agent/callback", type="jsonrpc", auth="public", methods=["POST"], csrf=False)
    def agent_event_callback(self, **kwargs: EventCallbackPayload):
        token = self._extract_api_key()
        agent = request.env["deploy.agent"].sudo().search([("api_key", "=", token)], limit=1)
        if not agent:
            return {"error": "Invalid API Key"}

        callback = EventCallbackPayload(**kwargs)
        event = agent.event_ids.filtered(lambda e: e.id == callback.event_id)

        if not event:
            _logger.warning(f"event not found for callback with event id: {callback.event_id}.")

        _logger.info(
            f"event response received: {callback.event_id}. status: {callback.status}. message: {callback.message}"
        )

        event.sudo().write({"status": callback.status, "message": callback.message})

        return {"status": "success"}

    @http.route(
        "/agent/get_script/<string:script_name>/<string:script_extension>",
        type="http",
        auth="public",
        methods=["GET"],
        csrf=False,
    )
    def get_script(self, script_name, script_extension):
        """Serve a script from the module directory"""

        filename = f"{script_name}"
        if script_extension != "0":
            filename += f".{script_extension}"

        module_path = get_module_path("deploy")
        script_path = os.path.join(module_path, "agent_scripts", filename)

        if not os.path.exists(script_path):
            return http.Response("Script not found", status=404)

        with open(script_path) as f:
            script_content = f.read()

        return http.Response(
            script_content,
            mimetype="text/plain",
            headers=[("Content-Disposition", f'attachment; filename="{filename}"')],
        )

    def _apply_heartbeat(self, agent, heartbeat_payload: HeartbeatPayload):
        agent.sudo().write(
            {
                "last_heartbeat": fields.Datetime.now(),
                "heartbeat_payload": heartbeat_payload.model_dump_json(),
            }
        )
