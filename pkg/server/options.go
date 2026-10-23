package server

import (
	"github.com/tyhal/hytale/pkg/auth"
)

type Options func(*serverOptions)

type serverOptions struct {
	backups       string
	sessionToken  auth.SessionToken
	idToken       auth.IdentityToken
	owner         auth.OwnerUUID
	ipv6         bool
	exec         bool
	service      bool
	javaFlags    []string
}

func WithBackups(backups string) Options {
	return func(opts *serverOptions) {
		opts.backups = backups
	}
}

func WithSessionToken(token auth.SessionToken) Options {
	return func(opts *serverOptions) {
		opts.sessionToken = token
	}
}

func WithIdentityToken(token auth.IdentityToken) Options {
	return func(opts *serverOptions) {
		opts.idToken = token
	}
}

func WithOwner(owner auth.OwnerUUID) Options {
	return func(opts *serverOptions) {
		opts.owner = owner
	}
}

func WithIPv6() Options {
	return func(opts *serverOptions) {
		opts.ipv6 = true
	}
}

func WithJavaFlags(flags ...string) Options {
	return func(opts *serverOptions) {
		opts.javaFlags = append(opts.javaFlags, flags...)
	}
}

func WithExitOnOOM() Options {
	return func(opts *serverOptions) {
		opts.javaFlags = append(opts.javaFlags, "-XX:+ExitOnOutOfMemoryError")
	}
}

func WithExec() Options {
	return func(opts *serverOptions) {
		opts.exec = true
	}
}

func WithService() Options {
	return func(opts *serverOptions) {
		opts.service = true
	}
}
