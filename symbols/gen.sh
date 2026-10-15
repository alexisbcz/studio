#!/bin/sh
# Regenerates the interpreter's symbol tables: the shadcn.go packages and
# the standard library packages programs may import.
#   go install github.com/traefik/yaegi/cmd/yaegi@<version in go.mod>
#   symbols/gen.sh
set -eu
cd "$(dirname "$0")"
find . -name '*.go' ! -name symbols.go -delete
pkgs=$(go list github.com/alexisbcz/components/... github.com/alexisbcz/libhtml github.com/alexisbcz/libhtml/secure github.com/alexisbcz/icons/lucide github.com/alexisbcz/icons/tabler | grep -v 'internal\|style/gen\|/runtime$')
std="reflect fmt strings strconv math math/rand math/rand/v2 time sort slices maps errors unicode unicode/utf8 unicode/utf16 encoding/json net/http net/url html regexp bytes cmp io iter context"
for p in $pkgs $std; do yaegi extract -name symbols "$p"; done
# encoding/json's Options interface has a method taking an internal type:
# drop its wrapper (programs cannot implement it anyway).
python3 - <<'PY'
import re
p = "encoding-json.go"
s = open(p).read()
s = s.replace('\t"encoding/json/internal"\n', '')
s = re.sub(r'\t\t"_Options":.*\n', '', s)
s = re.sub(r'// _encoding_json_Options is an interface wrapper.*?\n}\n\nfunc \(W _encoding_json_Options\).*?\n}\n\n', '', s, flags=re.S)
open(p, "w").write(s)
PY
gofmt -w .
