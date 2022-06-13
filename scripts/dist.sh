#!/usr/bin/env bash
set -e

# WARN: non hermetic build (people must run this script inside docker to
# produce deterministic binaries).

if [ -z "$VERSION" ]; then
	echo "Please specify a version."
	exit 1
fi
echo "==> Building version $VERSION..."

# Delete the old dir
echo "==> Removing old directory..."
rm -rf build/pkg
mkdir -p build/pkg

GIT_IMPORT="github.com/line/ostracon/version"

# Determine the arch/os combos we're building for
XC_ARCH=${XC_ARCH:-"386 amd64 arm"}
XC_OS=${XC_OS:-"solaris darwin freebsd linux windows"}
XC_EXCLUDE=${XC_EXCLUDE:-" darwin/arm solaris/amd64 solaris/386 solaris/arm freebsd/amd64 windows/arm linux/arm "}

# Make sure build tools are available.
#make tools # XXX Should remove "make tools": https://github.com/line/ostracon/commit/c6e0d20d4bf062921fcc1eb5b2399447a7d2226e#diff-76ed074a9305c04054cdebb9e9aad2d818052b07091de1f20cad0bbac34ffb52

# Build!
# ldflags: -s Omit the symbol table and debug information.
#	         -w Omit the DWARF symbol table.
echo "==> Building..."
IFS=' ' read -ra arch_list <<< "$XC_ARCH"
IFS=' ' read -ra os_list <<< "$XC_OS"
for arch in "${arch_list[@]}"; do
	for os in "${os_list[@]}"; do
		if [[ "$XC_EXCLUDE" !=  *" $os/$arch "* ]]; then
			echo "--> $os/$arch"
			GOOS=${os} GOARCH=${arch} go build -ldflags "-s -w -X ${GIT_IMPORT}.OCCoreSemVer=${VERSION}" -tags="${BUILD_TAGS}" -o "build/pkg/${os}_${arch}/ostracon" ./cmd/ostracon
		fi
	done
done

# Zip all the files.
echo "==> Packaging..."
for PLATFORM in $(find ./build/pkg -mindepth 1 -maxdepth 1 -type d); do
	OSARCH=$(basename "${PLATFORM}")
	echo "--> ${OSARCH}"

	pushd "$PLATFORM" >/dev/null 2>&1
	zip "../${OSARCH}.zip" ./*
	popd >/dev/null 2>&1
done

# Add "ostracon" and $VERSION prefix to package name.
rm -rf ./build/dist
mkdir -p ./build/dist
for FILENAME in $(find ./build/pkg -mindepth 1 -maxdepth 1 -type f); do
	FILENAME=$(basename "$FILENAME")
	cp "./build/pkg/${FILENAME}" "./build/dist/ostracon_${VERSION}_${FILENAME}"
done

# Make the checksums.
pushd ./build/dist
shasum -a256 ./* > "./ostracon_${VERSION}_SHA256SUMS"
popd

# Done
echo
echo "==> Results:"
ls -hl ./build/dist

exit 0
