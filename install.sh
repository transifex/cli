#!/bin/bash

#
# Script definetely inspired by
#
# https://github.com/nvm-sh/nvm/blob/master/install.sh
# https://github.com/heroku/cli/blob/master/install-standalone.sh
#

{
    echoerr() { echo "$@" 1>&2; }

    try_profile() {
        if [ -z "${1-}" ] || [ ! -f "${1}" ]; then
            return 1
        else
            echo $1
        fi
    }

    find_profile() {
        local DETECTED_PROFILE
        if [ "${SHELL#*bash}" != "$SHELL" ]; then
            if [ -f "$HOME/.bashrc" ]; then
                DETECTED_PROFILE="$HOME/.bashrc"
            elif [ -f "$HOME/.bash_profile" ]; then
                DETECTED_PROFILE="$HOME/.bash_profile"
            fi
        elif [ "${SHELL#*zsh}" != "$SHELL" ]; then
            if [ -f "$HOME/.zshrc" ]; then
                DETECTED_PROFILE="$HOME/.zshrc"
            fi
        fi

        if [ -z "$DETECTED_PROFILE" ]; then
            for EACH_PROFILE in ".profile" ".bashrc" ".bash_profile" ".zshrc"; do
                if DETECTED_PROFILE="$(try_profile "${HOME}/${EACH_PROFILE}")"; then
                    break
                fi
            done
        fi

        if [ -n "$DETECTED_PROFILE" ]; then
            echo "$DETECTED_PROFILE"
        else
            return 1
        fi
    }

    if [ -f tx ]; then
        echo "Filename 'tx' already exists. Please remove it or try running the script from a different folder"
        exit 1
    fi

    # Get the latest version as string from Github API
    LATEST_URL="https://api.github.com/repos/transifex/cli/releases/latest"
    LATEST_VERSION="$(curl -s "$LATEST_URL" | grep "tag_name" |
        cut -d : -f 2 | tr -d \"\,\ )"

    # Try to figure out the version needed to download for the specific system

    if [ "$(uname)" == "Darwin" ]; then
        OS=darwin
    elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
        OS=linux
    else
        echoerr "This installer is only supported on Linux and MacOS"
        exit 1
    fi

    ARCH="$(uname -m)"
    if [ "$ARCH" == "x86_64" ]; then
        ARCH=amd64
    elif [[ "$ARCH" == "aarch"* ]] || [[ "$ARCH" == "arm"* ]]; then
        ARCH=arm64
    elif [[ "$ARCH" == "i386" ]]; then
        ARCH=386
    else
        echoerr "unsupported arch: $ARCH"
        exit 1
    fi

    # Try to download the version from github releases
    URL="https://github.com/transifex/cli/releases/download/$LATEST_VERSION/tx-$OS-$ARCH.tar.gz"

    echo -e "** Installing CLI from $URL\n"
    if tar --version | grep -q 'gnu'
    then
        curl -L "$URL" | tar xz --skip-old-files
    else
        curl -L "$URL" | tar kxz
    fi

    # Try to add tx to PATH
    echo -e "\n** Adding CLI to PATH"
    DETECTED_PROFILE=$(find_profile)

    if [ -n "$DETECTED_PROFILE" ]; then
        echo "export PATH=\"$PWD:\$PATH\"" >> "$DETECTED_PROFILE"
        echo "** export PATH=\"$PWD:\$PATH\" was added to $DETECTED_PROFILE. Please restart your terminal to have access to 'tx' from any path."
    else
        echo "** Profile not found, we tried for .profile, .bashrc, .bash_profile, .zshrc"
        echo "** Please add this line to the correct file if you want to access 'tx' from any path."
        echo -e "\nexport PATH=\"$PWD:\$PATH\"\n"
    fi

    echo -e "\n** If everything went fine you should see the Transifex CLI version in the following line."
    ./tx -v
}
