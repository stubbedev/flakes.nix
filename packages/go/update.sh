# shellcheck shell=bash
dir="$1"
file="$dir/hashes.json"

current="$(jq -er '.version' "$file")"

release="$(curl -fsSL --max-time 60 --retry 3 'https://go.dev/dl/?mode=json' | jq -ec 'map(select(.stable)) | first')"
latest="$(jq -er '.version | ltrimstr("go")' <<<"$release")"

if [[ "$latest" == "$current" ]]; then
  echo "go: $current is up to date" >&2
  exit 0
fi

newest="$(printf '%s\n%s\n' "$current" "$latest" | sort -V | tail -n 1)"
if [[ "$newest" != "$latest" ]]; then
  echo "go: upstream latest $latest is older than current $current" >&2
  exit 1
fi

hex_hashes="$(
  jq -ec --slurpfile platforms "$GO_PLATFORMS_FILE" '
    .files as $files
    | $platforms[0]
    | map_values(. as $platform
        | [$files[] | select(.kind == "archive" and "\(.os)-\(.arch)" == $platform)]
        | if length == 1 then .[0].sha256
          else error("go \($release_version): expected one archive for \($platform), found \(length)") end)
  ' --arg release_version "$latest" <<<"$release"
)"

hashes='{}'
while IFS=$'\t' read -r system hex; do
  sri="sha256-$(tr 'a-f' 'A-F' <<<"$hex" | basenc --base16 -d | base64 -w 0)"
  hashes="$(jq -c --arg system "$system" --arg sri "$sri" '.[$system] = $sri' <<<"$hashes")"
done < <(jq -r 'to_entries[] | "\(.key)\t\(.value)"' <<<"$hex_hashes")

tmp="$(mktemp "$file.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
jq -n --arg version "$latest" --argjson hashes "$hashes" '{version: $version, hashes: $hashes}' >"$tmp"
mv "$tmp" "$file"
trap - EXIT

echo "go: $current -> $latest"
