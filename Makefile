NAME    = banana
SRC     = ./src
DIST    = ./dist
SKILL   = ./skill
LDFLAGS = -s -w

# All platform targets
PLATFORMS = \
	linux-amd64 \
	linux-arm64 \
	darwin-amd64 \
	darwin-arm64 \
	windows-amd64 \
	windows-arm64

# Build for the current platform (development use)
build:
	cd $(SRC) && CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o ../$(NAME)

# Build all platform zips
release: clean $(addprefix zip-,$(PLATFORMS))

# Pattern: zip-<os>-<arch>
# Splits the target name on '-' to extract GOOS and GOARCH,
# builds the binary, stages it with the skill files, and zips.
zip-%:
	$(eval OS   = $(word 1,$(subst -, ,$*)))
	$(eval ARCH = $(word 2,$(subst -, ,$*)))
	$(eval BIN  = $(if $(filter windows,$(OS)),$(NAME).exe,$(NAME)))
	@mkdir -p $(DIST)
	cd $(SRC) && GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 \
		go build -ldflags="$(LDFLAGS)" -o ../$(DIST)/staging/$(NAME)/$(BIN)
	cp $(SKILL)/SKILL.md $(SKILL)/cli-reference.md $(SKILL)/prompting-reference.md $(DIST)/staging/$(NAME)/
	cd $(DIST)/staging && zip -r ../$(NAME)-$(OS)-$(ARCH).zip $(NAME)/
	rm -rf $(DIST)/staging

clean:
	rm -rf $(DIST) $(NAME) $(NAME).exe

.PHONY: build release clean
