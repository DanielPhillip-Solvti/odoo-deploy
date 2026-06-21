{
    'name': 'Deploy',
    'summary': 'Module for deploying Odoo on a server.',
    'version': '19.0.0.0.1',
    'category': 'Tools',
    'data': [
        'security/ir.model.access.csv',
        'views/deploy_environment_views.xml',
        'views/deploy_vm_views.xml',
        'views/deploy_menu_items.xml',
        "views/github_app_config_views.xml",
        'data/github_app_config.xml',
    ],
    'assets': {
        'web.assets_backend': [
            'deploy/static/src/components/github_commit_history/github_commit_history.js',
            'deploy/static/src/components/github_commit_history/github_commit_history.xml',
        ],
    },
    'depends':[
        "obscure_field"
    ],
    'installable': True,
    'application': True,
}