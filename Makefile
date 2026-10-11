SHELL := /usr/bin/env bash
GOLIB ?= golib

.PHONY: check check-full ci cohesion inventory repository-check

check:
	$(GOLIB) check --local --all

check-full:
	$(GOLIB) check --all

ci: repository-check check

cohesion:
	$(GOLIB) cohesion check

inventory repository-check:
	$(GOLIB) repository check
