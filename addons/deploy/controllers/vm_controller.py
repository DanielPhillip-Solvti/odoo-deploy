import os

from odoo import _, http
from odoo.exceptions import UserError
from odoo.http import request
from odoo.modules.module import get_module_path


class VMController(http.Controller):
    @http.route("/register_agent", type="jsonrpc", auth="public")
    def register_agent(self, **kwargs):
        vm_id = kwargs.get("vm_id", False)
        otp = kwargs.get("otp", False)
        ip_address = kwargs.get("ip_address", False)

        if not (vm_id and otp and ip_address):
            raise UserError(_("vm_id, otp and ip address must be provided to allow an agent registration"))

        vm = request.env["deploy.vm"].browse(vm_id)
        if not vm:
            raise UserError(_("vm with id %s not found", vm_id))

        return vm.sudo().exchange_otp(otp, ip_address)

    def _get_agent_script_file(self, script_name):
        """Helper method to retrieve a script file from the module directory."""
        module_path = get_module_path("deploy")
        script_path = os.path.join(module_path, "agent_scripts", script_name)

        if not os.path.exists(script_path):
            return None

        with open(script_path) as f:
            return f.read()

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
