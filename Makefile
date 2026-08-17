BINARY        = terraform-provider-apiname
INSTALL_DIR   = $(HOME)/.terraform.d/plugins/registry.terraform.io/abtme/apiname/0.1.0/linux_amd64

.PHONY: build install clean

build:
	go build -o $(BINARY) .

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

clean:
	rm -f $(BINARY)
