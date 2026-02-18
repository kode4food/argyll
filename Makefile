SUBPROJECTS = engine web mcp archiver sdks/go-builder sdks/python \
	examples/inventory-resolver examples/notification-sender \
	examples/order-creator examples/simple-step \
	examples/stock-reservation examples/user-resolver

.PHONY: all install build format check test pre-commit clean

define run_target
	@for dir in $(SUBPROJECTS); do \
		if $(MAKE) -C $$dir -n $(1) >/dev/null 2>&1; then \
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

check:
	$(call run_target,check)

test:
	$(call run_target,test)

pre-commit:
	$(call run_target,pre-commit)

clean:
	$(call run_target,clean)
