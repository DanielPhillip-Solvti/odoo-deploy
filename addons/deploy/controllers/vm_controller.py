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

        try:
            request.env["bus.bus"]._sendone(
                f"deploy_agent_{agent.id}",
                "deploy.event_callback",
                {
                    "event_id": callback.event_id,
                    "status": callback.status,
                    "message": callback.message,
                    "branch": event.parameters.get("branch") if event.parameters else None,
                },
            )
        except Exception:
            _logger.warning("Failed to broadcast event callback via bus", exc_info=True)

        return {"status": "success"}

    @http.route("/agent/update_config", type="jsonrpc", auth="public", methods=["POST"], csrf=False)
    def agent_update_config(self, **kwargs):
        token = self._extract_api_key()
        agent = request.env["deploy.agent"].sudo().search([("api_key", "=", token)], limit=1)
        if not agent:
            return {"error": "Invalid API Key"}
        repo_url = kwargs.get("repository_url")
        if repo_url:
            agent.sudo().write({"repository_url": repo_url})
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

    @http.route("/agent/ws/token", type="jsonrpc", auth="user", methods=["POST"], csrf=False)
    def request_ws_token(self, **kwargs):
        agent_id = kwargs.get("agent_id")
        purpose = kwargs.get("purpose")
        params = kwargs.get("params", {})
        if not agent_id or not purpose:
            return {"error": "Missing agent_id or purpose"}

        agent = request.env["deploy.agent"].browse(agent_id)
        if not agent.exists():
            return {"error": "Agent not found"}

        return agent.request_ws_token(purpose, params)

    @http.route("/agent/validate_ws_token", type="jsonrpc", auth="public", methods=["POST"], csrf=False)
    def validate_ws_token(self, **kwargs):
        api_key = self._extract_api_key()
        agent = request.env["deploy.agent"].sudo().search([("api_key", "=", api_key)], limit=1)
        if not agent:
            return {"valid": False, "purpose": "", "params": {}}

        token_value = kwargs.get("token")
        if not token_value:
            return {"valid": False, "purpose": "", "params": {}}

        ws_token = request.env["deploy.ws_token"].sudo().search([("token", "=", token_value)], limit=1)
        if not ws_token or not ws_token.is_valid():
            return {"valid": False, "purpose": "", "params": {}}

        ws_token.mark_used()
        return {"valid": True, "purpose": ws_token.purpose, "params": ws_token.params}

    # Backward-compat wrappers -------------------------------------------------

    @http.route("/agent/backup/token", type="jsonrpc", auth="user", methods=["POST"], csrf=False)
    def request_backup_token(self, **kwargs):
        agent_id = kwargs.get("agent_id")
        filename = kwargs.get("filename")
        if not agent_id or not filename:
            return {"error": "Missing agent_id or filename"}
        agent = request.env["deploy.agent"].browse(agent_id)
        if not agent.exists():
            return {"error": "Agent not found"}
        return agent.request_download_token(filename)

    @http.route("/agent/logs/token", type="jsonrpc", auth="user", methods=["POST"], csrf=False)
    def request_log_token(self, **kwargs):
        agent_id = kwargs.get("agent_id")
        branch = kwargs.get("branch")
        if not agent_id or not branch:
            return {"error": "Missing agent_id or branch"}
        agent = request.env["deploy.agent"].browse(agent_id)
        if not agent.exists():
            return {"error": "Agent not found"}
        return agent.request_log_token(branch)

    @http.route("/agent/validate_token", type="jsonrpc", auth="public", methods=["POST"], csrf=False)
    def validate_download_token(self, **kwargs):
        api_key = self._extract_api_key()
        agent = request.env["deploy.agent"].sudo().search([("api_key", "=", api_key)], limit=1)
        if not agent:
            return {"valid": False, "filename": ""}
        download_token_value = kwargs.get("token")
        if not download_token_value:
            return {"valid": False, "filename": ""}
        download_token = (
            request.env["deploy.download_token"].sudo().search([("token", "=", download_token_value)], limit=1)
        )
        if not download_token or not download_token.is_valid():
            return {"valid": False, "filename": ""}
        download_token.mark_used()
        return {"valid": True, "filename": download_token.filename}

    @http.route("/agent/validate_log_token", type="jsonrpc", auth="public", methods=["POST"], csrf=False)
    def validate_log_token(self, **kwargs):
        api_key = self._extract_api_key()
        agent = request.env["deploy.agent"].sudo().search([("api_key", "=", api_key)], limit=1)
        if not agent:
            return {"valid": False, "branch": ""}
        token_value = kwargs.get("token")
        if not token_value:
            return {"valid": False, "branch": ""}
        stream_token = request.env["deploy.stream_token"].sudo().search([("token", "=", token_value)], limit=1)
        if not stream_token or not stream_token.is_valid():
            return {"valid": False, "branch": ""}
        stream_token.mark_used()
        return {"valid": True, "branch": stream_token.branch}

    def _apply_heartbeat(self, agent, heartbeat_payload: HeartbeatPayload):
        vals = {
            "last_heartbeat": fields.Datetime.now(),
            "heartbeat_payload": heartbeat_payload.model_dump(),
        }
        if heartbeat_payload.repo_url:
            vals["repository_url"] = heartbeat_payload.repo_url
        if heartbeat_payload.ws_url:
            vals["ws_url"] = heartbeat_payload.ws_url
        agent.sudo().write(vals)
        agent.sudo()._broadcast_heartbeat_via_bus()
