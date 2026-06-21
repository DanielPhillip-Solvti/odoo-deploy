from odoo import http, _
from odoo.exceptions import UserError
from odoo.http import request, Response

class VMController(http.Controller):
    @http.route('/register_agent', type='jsonrpc', auth='public')
    def register_agent(self, **kwargs):
        vm_id = kwargs.get('vm_id', False)
        otp = kwargs.get('otp', False)
        ip_address = kwargs.get('ip_address', False)
        if not (vm_id and otp and ip_address):
            raise UserError(_("vm_id, otp and ip address must be provided to allow an agent registration"))
        vm = request.env["deploy.vm"].browse(vm_id)
        if not vm:
            raise UserError(_("vm with id %{id}s not found", id=vm_id))
        return vm.sudo().exchange_otp(otp, ip_address)

    @http.route('/agent/install/<string:otp>', type='http', auth='public', methods=['GET'])
    def install_agent(self, otp, **kwargs):
        vm = request.env["deploy.vm"].sudo().search([('otp', '=', otp)], limit=1)
        if not vm:
            return Response("Invalid or expired install token", status=404)

        return Response("not implemnented yet", content_type='text/plain')
