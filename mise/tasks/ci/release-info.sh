#!/usr/bin/env bash
#MISE description="Generate release info for the GitHub workflow"
set -euo pipefail

gh_output() {
	local key value
	key="$1"
	value="$2"

	if [[ -z "$GITHUB_OUTPUT" ]]; then
		echo "output: $key => $value"
		return
	fi

	if [[ -z "$key" ]]; then
		echo "output: missing key " >&2
	elif [[ -z "$value" ]]; then
		echo "output: missing value for key '$key'" >&2
	else
		{
			echo "$key<<EOF"
			echo "$value"
			echo "EOF"
		} >>"$GITHUB_OUTPUT"
	fi
}

gh_repository="${1:-$GITHUB_REPOSITORY}"
gh_token="${2:-$GITHUB_TOKEN}"
gh_event_name="$GITHUB_EVENT_NAME"
gh_pr_number="${GITHUB_REF_NAME%%/merge}"
version="$(git-cliff --bumped-version)"

changelog="$(
	git-cliff \
		--tag "$version" \
		--github-repo "$gh_repository" \
		--github-token "$gh_token" \
		--unreleased \
		--strip all \
		-v
)"

gh_output "version" "$version"
gh_output "changelog" "$changelog"

if [[ "$gh_event_name" != "pull_request" ]]; then
	exit 0
fi

if [[ -n "$changelog" ]]; then
	pr_comment=$(
		cat <<-EOF
			# :pencil: Changelog preview
			Below is a preview of the Changelog that will be added to the next release. Only commit messages that follow the [Conventional Commits specification](https://www.conventionalcommits.org/) will be included in the Changelog.

			$changelog
		EOF
	)
else
	pr_comment=$(
		cat <<-EOF
			# :disappointed: No release for you
			There are no commits in your branch that follow the [Conventional Commits specification](https://www.conventionalcommits.org/), so no release will be created.

			If you want to create a release from this pull request, please reword your commit messages to replace this message with a preview of a beautiful Changelog."
		EOF
	)
fi

gh pr comment "$gh_pr_number" \
	--edit-last --create-if-none \
	--repo "$gh_repository" \
	--body "$pr_comment"
