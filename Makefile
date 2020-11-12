test:
	go test

fmt:
	gofmt -w -s -l .

doc:
	@echo "open http://localhost:6060/pkg/github.com/rsms/go-json/"
	@bash -c '[ "$$(uname)" == "Darwin" ] && \
	         (sleep 1 && open "http://localhost:6060/pkg/github.com/rsms/go-json/") &'
	godoc -http=localhost:6060

clean:
	rm -rvf "$(GOCOV_HTML_FILE)" "$(CACHE_DIR)"

.PHONY: test fmt doc clean
