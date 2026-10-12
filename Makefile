SHELL := /usr/bin/env bash
GOLIB ?= golib

.PHONY: source-docs-check check check-full ci cohesion inventory repository-check

check: source-docs-check
	$(GOLIB) check --local --all

check-full: source-docs-check
	$(GOLIB) check --all

ci: source-docs-check repository-check check

cohesion:
	$(GOLIB) cohesion check

inventory repository-check:
	$(GOLIB) repository check

source-docs-check:
	$(GOLIB) source-docs check
