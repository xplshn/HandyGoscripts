#!/bin/sh
#set -x

# shellcheck disable=SC2031
# shellcheck disable=SC2030

OPWD="$PWD"
BASE="$(dirname "$(realpath "$0")")"
if [ "$OPWD" != "$BASE" ]; then
    echo "... $BASE is not the same as $PWD ..."
    echo "Going into $BASE and coming back here in a bit"
    cd "$BASE" || exit 1
fi
trap 'cd "$OPWD"' EXIT

# Function to log to stdout with green color
log() {
    _Xashstd_reset="\033[m"
    _Xashstd_color_code="\033[32m"
    printf "${_Xashstd_color_code}->${_Xashstd_reset} %s\n" "$*"
}

# Function to log_warning to stdout with yellow color
log_warning() {
    _Xashstd_reset="\033[m"
    _Xashstd_color_code="\033[33m"
    printf "${_Xashstd_color_code}->${_Xashstd_reset} %s\n" "$*"
}

# Function to log_error to stdout with red color
log_error() {
    _Xashstd_reset="\033[m"
    _Xashstd_color_code="\033[31m"
    printf "${_Xashstd_color_code}->${_Xashstd_reset} %s\n" "$*"
    exit 1
}

# Unified function to build Go commands or execute cbuild.sh in the specified directories
build_commands() {
    for main_dir in "$@"; do
        if [ -d "$main_dir" ]; then
            log "Processing directory: $main_dir"
            # Process directories containing .go files or cbuild.sh
            find "$main_dir" -type d | while IFS= read -r dir; do
                if [ -d "$dir" ]; then
                    # Check for .go files in the current directory
                    if find "$dir" -maxdepth 1 -type f -name '*.go' -print -quit | grep -q .; then
                        log "Building Go projects in $dir"
                        (cd "$dir" && go build)
                    fi

                    # Check for cbuild.sh in the current directory
                    if [ -f "$dir/cbuild.sh" ]; then
                        log "Building C project in $dir"
                        (cd "$dir" && ./cbuild.sh)
                    fi
                elif [ "$main_dir" != "" ]; then
                    log_warning "Directory \"$dir\" does not exist"
                fi
            done
        elif [ "$main_dir" != "" ]; then
            log_warning "Directory \"$main_dir\" does not exist"
        fi
    done
}

# Function to copy built executables to the built directory
copy_executables() {
    mkdir -p ./built
    # Find all Go files and extract their directories
    find "$@" -type f -name '*.go' -print | while IFS= read -r go_file; do
        dir=$(dirname "$go_file")
        # Check if the directory contains Go files
        if [ "$(find "$dir" -type f -name '*.go' -print -quit)" ]; then
            # Copy only the executables from directories that contain Go files
            find "$dir" -type f -executable -exec mv {} ./built/ \;
        fi
    done
    # Find all directories containing a cbuild.sh script
    find "$@" -type f -name 'cbuild.sh' -print | while IFS= read -r cbuild_file; do
        dir=$(dirname "$cbuild_file")
        # Copy only the executables from directories that contain a cbuild.sh script
        find "$dir" -type f -executable -exec mv {} ./built/ \;
    done
}

# Function to clean up the built directory and remove binaries in specified main directories
clean_up() {
    # Remove the built directory
    rm -rf ./built >/dev/null 2>&1

    # Process each main directory
    for main_dir in "$@"; do
        if [ -d "$main_dir" ]; then
            log "Processing directory: $main_dir"
            # Find all directories containing Go files or a cbuild.sh script
            find "$main_dir" -type d | while IFS= read -r dir; do
                if [ "$(find "$dir" -maxdepth 1 -type f -name '*.go' -print -quit)" ] || [ -f "$dir/cbuild.sh" ]; then
                    log "Checking directory: $dir"
                    # Remove only the executables from directories that contain Go files or cbuild.sh script
                    log "Removing executables in directory: $dir"
                    find "$dir" -type f -executable -exec rm {} +
                    # Execute cbuild.sh clean if it exists
                    if [ -f "$dir/cbuild.sh" ]; then
                        log "Cleaning C project in $dir"
                        (cd "$dir" && ./cbuild.sh clean)
                    fi
                fi
            done
        elif [ "$main_dir" != "" ]; then
            log_warning "Directory \"$main_dir\" does not exist"
        fi
    done
}

# Function to update dependencies for Go projects
update_go_deps() {
    for dir in "$@"; do
        if [ -d "$dir" ]; then
            log "Updating Go deps for: $dir"
            (cd "$dir" && go get -u && go mod tidy)
        fi
    done
}

# Function to read extend*.b files and set repository variables
read_extendFiles() {
    for file in "$@"; do
        if [ -f "$file" ]; then
            while IFS= read -r line; do
                REPO_DEST="$(echo "$line" | cut -d ' ' -f 1)"
                REPO_URL="$(echo "$line" | awk '{print $NF}')"
                # Export the variables to be used by clone_repos
                export REPO_DEST REPO_URL
                echo "$REPO_DEST $REPO_URL" >>/tmp/repo_list
            done <"$file"
        fi
    done
}

# Function to clone repositories from given files or update if they already exist
clone_repos() {
    target_dir=$1
    shift
    mkdir -p "$target_dir"
    # Read repositories from the list created by read_extendFiles
    if [ -f /tmp/repo_list ]; then
        while IFS= read -r repo_info; do
            REPO_DEST=$(echo "$repo_info" | cut -d ' ' -f 1)
            REPO_URL=$(echo "$repo_info" | awk '{print $NF}')
            repo_path="$target_dir/$REPO_DEST"
            if [ -d "$repo_path/.git" ]; then
                log "Updating $REPO_URL in $repo_path"
                (cd "$repo_path" && git pull)
            else
                log "Cloning $REPO_URL into $repo_path"
                git clone --depth 1 "$REPO_URL" "$repo_path"
            fi
        done </tmp/repo_list
    fi
    rm /tmp/repo_list >/dev/null 2>&1
}

# Main script execution
case "$1" in
"" | "build")
    log "Starting build process"
    read_extendFiles ./extendGo.b ./extendC.b
    clone_repos "./extend"
    build_commands "./cmd" "./extend"
    copy_executables "./cmd" "./extend"
    log "Build process completed"
    ;;
"clean")
    shift
    log "Starting clean process"
    clean_up "$2" "./cmd" "./extend"
    log "Clean process completed"
    ;;
"deps")
    log "Updating dependencies"
    update_go_deps "./cmd" "./extendGo"
    log "Dependencies updated"
    ;;
*)
    echo "Usage: $0 {build|clean|deps}"
    exit 1
    ;;
esac