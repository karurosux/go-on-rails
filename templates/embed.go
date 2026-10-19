// Package templates embeds every code template gor renders. Templates are
// plain files with a .tmpl suffix; edit them to change what gor generates.
package templates

import "embed"

//go:embed all:project all:resource all:migration all:auth all:jobs all:realtime all:mailer all:uploads all:authemails all:skill all:observability all:plugindev
var FS embed.FS
