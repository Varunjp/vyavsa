package web

import "embed"

// WebFS embeds all static web assets and HTML templates for production deployment
//
//go:embed templates/* static/*
var WebFS embed.FS
