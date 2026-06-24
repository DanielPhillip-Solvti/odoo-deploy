package helpers

import (
	"agent/config"
	"fmt"
)

const envDockerfileBase = `ARG ODOO_VERSION
FROM ghcr.io/solvti/odoo:${ODOO_VERSION}
`

const envDockerfileWithRequirements = `ARG ODOO_VERSION
FROM ghcr.io/solvti/odoo:${ODOO_VERSION}

COPY ./addons/requirements.txt /tmp/requirements.txt

USER root

RUN pip install --no-cache-dir -r /tmp/requirements.txt

USER odoo
`

func EnvDockerfile(hasRequirements bool) string {
	if hasRequirements {
		return envDockerfileWithRequirements
	}
	return envDockerfileBase
}

func OdooConf(branch string) string {
	return fmt.Sprintf(`[options]
db_host = db
db_user = odoo
db_password = odoo
db_name = %s
addons_path = %s,/code/enterprise,/code/odoo/addons
`, branch, config.AddonsPathContainer)
}
