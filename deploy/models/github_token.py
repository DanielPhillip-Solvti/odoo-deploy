from odoo import models, fields, api


class DeployGithubToken(models.Model):
    _name = 'deploy.github.token'
    _description = 'GitHub OAuth Token per User'

    user_id = fields.Many2one('res.users', required=True, ondelete='cascade', index=True)
    access_token = fields.Char(string='Access Token')

    _sql_constraints = [
        ('user_unique', 'unique(user_id)', 'Only one GitHub token per user'),
    ]

    @api.model
    def get_token_for_current_user(self):
        record = self.sudo().search([('user_id', '=', self.env.uid)], limit=1)
        return record.access_token if record else False

    @api.model
    def set_token_for_current_user(self, token):
        existing = self.sudo().search([('user_id', '=', self.env.uid)], limit=1)
        if existing:
            existing.access_token = token
        else:
            self.sudo().create({'user_id': self.env.uid, 'access_token': token})

    @api.model
    def clear_token_for_current_user(self):
        self.sudo().search([('user_id', '=', self.env.uid)]).unlink()
