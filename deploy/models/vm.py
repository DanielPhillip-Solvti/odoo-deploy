from odoo import models, fields, api
from ..services.agent_service import AgentService
import random
import string

class VM(models.Model):
    _name = 'deploy.vm'
    _description = 'Virtual Machine for Odoo Deployment'
    
    name = fields.Char(string='VM Name', required=True)
    url = fields.Char(string='URL', readonly=False)
    repository_url = fields.Char(string='Repository URL', required=True, help='Git repository URL for the Odoo deployment including branch or tag if necessary')
    environment_ids = fields.One2many('deploy.environment', 'vm_id', string='Environments')
    otp = fields.Char(string='One-Time Password', readonly=True)
    capabilities = fields.Json(help='Commands and functions that the VM can execute')
    status = fields.Selection(
        [('pending', 'Pending'), ('active', 'Active'), ('error', 'Error')], 
        default='pending',
        readonly=True
    )

    def _generate_otp(self):
        return ''.join(random.choices(string.ascii_uppercase + string.digits, k=6))
    
    def exchange_otp(self, provided_otp, url):
        self.ensure_one()
        if self.otp == provided_otp:
            self.otp = False
            self.url = url
            self.status = 'active'
            return True
        return False

    @api.model
    def create(self, vals_list):
        for vals in vals_list:
            vals['otp'] = self._generate_otp()
        return super().create(vals_list)
    
    def check_health(self):
        return AgentService().check_health(self)

    def backup(self, with_dump=False):
        return AgentService().backup(self, with_dump=with_dump)
    
    def download_dump(self, production):
        return AgentService().download_dump(self, production=production)

    # buttons
    def backup_no_dump(self):
        return self.backup(with_dump=False)

    def backup_with_dump(self):
        return self.backup(with_dump=True)

    def download_production_dump(self):
        return self.download_dump(production=True)

    def download_neutralised_dump(self):
        return self.download_dump(production=False)