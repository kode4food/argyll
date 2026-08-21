SUBPROJECTS = engine web mcp sdk/go sdk/python \
	examples/inventory-resolver examples/notification-sender \
	examples/order-creator examples/simple-step \
	examples/stock-reservation examples/user-resolver

.PHONY: all install build format generate check test pre-commit clean

define run_target
	@set -e; \
	for dir in $(SUBPROJECTS); do \
		if [ -f $$dir/Makefile ] && \
			grep -Eq '(^|[[:space:]])$(1)([[:space:]].*)?:' $$dir/Makefile; then \
			$(MAKE) -C $$dir $(1); \
		else \
			echo "Skipping $$dir (no $(1) target)"; \
		fi; \
	done
endef

all:
	$(call run_target,all)

install:
	$(call run_target,install)

build:
	$(call run_target,build)

format:
	$(call run_target,format)

generate:
	$(call run_target,generate)

check:
	$(call run_target,check)

test:
	$(call run_target,test)

pre-commit:
	$(call run_target,pre-commit)

clean:
	$(call run_target,clean)
