#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
plans_dir="${repo_root}/docs/plans"

if [[ -z "${NO_COLOR:-}" && ( -t 1 || -n "${FORCE_COLOR:-}" ) ]]; then
    color_reset=$'\033[0m'
    color_bold=$'\033[1m'
    color_red=$'\033[31m'
    color_green=$'\033[32m'
    color_yellow=$'\033[33m'
    color_blue=$'\033[34m'
    color_magenta=$'\033[35m'
    color_dim=$'\033[2m'
else
    color_reset=""
    color_bold=""
    color_red=""
    color_green=""
    color_yellow=""
    color_blue=""
    color_magenta=""
    color_dim=""
fi

if [[ ! -d "${plans_dir}" ]]; then
    printf "%splans directory not found:%s %s\n" "${color_red}" "${color_reset}" "${plans_dir}" >&2
    exit 1
fi

colorize_status() {
    local status="$1"
    local padded="$2"

    case "${status}" in
    Draft) printf "%s%s%s" "${color_blue}" "${padded}" "${color_reset}" ;;
    Active) printf "%s%s%s" "${color_yellow}" "${padded}" "${color_reset}" ;;
    Paused) printf "%s%s%s" "${color_magenta}" "${padded}" "${color_reset}" ;;
    Done) printf "%s%s%s" "${color_green}" "${padded}" "${color_reset}" ;;
    Cancelled) printf "%s%s%s" "${color_dim}" "${padded}" "${color_reset}" ;;
    *) printf "%s%s%s" "${color_red}" "${padded}" "${color_reset}" ;;
    esac
}

colorize_progress() {
    local complete="$1"
    local total="$2"
    local padded="$3"

    if [[ "${total}" -eq 0 ]]; then
        printf "%s%s%s" "${color_dim}" "${padded}" "${color_reset}"
    elif [[ "${complete}" -eq "${total}" ]]; then
        printf "%s%s%s" "${color_green}" "${padded}" "${color_reset}"
    elif [[ "${complete}" -eq 0 ]]; then
        printf "%s%s%s" "${color_dim}" "${padded}" "${color_reset}"
    else
        printf "%s%s%s" "${color_yellow}" "${padded}" "${color_reset}"
    fi
}

read_org_metadata() {
    local plan="$1"

    awk '
      BEGIN { title = ""; status = "" }
      {
        lower = tolower($0)
        if (title == "" && lower ~ /^#\+title:[[:space:]]*/) {
          sub(/^#\+[Tt][Ii][Tt][Ll][Ee]:[[:space:]]*/, "", $0)
          title = $0
        } else if (status == "" && lower ~ /^#\+status:[[:space:]]*/) {
          sub(/^#\+[Ss][Tt][Aa][Tt][Uu][Ss]:[[:space:]]*/, "", $0)
          status = $0
        }
      }
      END { printf "%s\t%s\n", title, status }
    ' "${plan}"
}

read_markdown_metadata() {
    local plan="$1"

    awk '
      BEGIN {
        title = ""
        status = ""
        line_nr = 0
        in_front_matter = 0
      }
      {
        line_nr++

        if (line_nr == 1 && $0 == "---") {
          in_front_matter = 1
          next
        }

        if (in_front_matter) {
          if ($0 == "---") {
            in_front_matter = 0
            next
          }

          if (status == "" && tolower($0) ~ /^status:[[:space:]]*/) {
            value = $0
            sub(/^[^:]+:[[:space:]]*/, "", value)
            status = value
          }

          if (title == "" && tolower($0) ~ /^title:[[:space:]]*/) {
            value = $0
            sub(/^[^:]+:[[:space:]]*/, "", value)
            title = value
          }

          next
        }

        if (title == "" && $0 ~ /^# /) {
          title = $0
          sub(/^#[[:space:]]+/, "", title)
        }
      }
      END { printf "%s\t%s\n", title, status }
    ' "${plan}"
}

read_checklist_counts() {
    local plan="$1"

    awk '
      BEGIN { complete = 0; total = 0 }
      /^([[:space:]]*[-+*]|[[:space:]]*[0-9]+\.)[[:space:]]+\[[Xx ]\]/ {
        total++
        if ($0 ~ /\[[Xx]\]/) {
          complete++
        }
      }
      END { printf "%d %d\n", complete, total }
    ' "${plan}"
}

printf "%s%-4s  %-10s  %-9s  %s%s\n" "${color_bold}" "Plan" "Status" "Progress" "Title" "${color_reset}"
printf "%s%-4s  %-10s  %-9s  %s%s\n" "${color_dim}" "----" "------" "--------" "------------------------------" "${color_reset}"

failures=0
found=0

shopt -s nullglob
plans=("${plans_dir}"/*.org "${plans_dir}"/*.md)
shopt -u nullglob

if [[ "${#plans[@]}" -gt 0 ]]; then
    mapfile -t plans < <(printf '%s\n' "${plans[@]}" | sort -V)
fi

for plan in "${plans[@]}"; do
    found=1

    filename="$(basename "${plan}")"
    plan_id="${filename%%-*}"
    extension="${filename##*.}"

    case "${extension}" in
    org)
        metadata="$(read_org_metadata "${plan}")"
        ;;
    md)
        metadata="$(read_markdown_metadata "${plan}")"
        ;;
    *)
        continue
        ;;
    esac

    title="${metadata%%$'\t'*}"
    status="${metadata#*$'\t'}"

    counts="$(read_checklist_counts "${plan}")"
    complete_count="${counts%% *}"
    total_count="${counts##* }"

    if [[ -z "${title}" ]]; then
        title="(missing title)"
        failures=$((failures + 1))
    fi

    if [[ -z "${status}" ]]; then
        status="(missing)"
        failures=$((failures + 1))
    fi

    status_field="$(printf "%-10s" "${status}")"
    progress_field="$(printf "%-9s" "${complete_count}/${total_count}")"
    progress_text="$(colorize_progress "${complete_count}" "${total_count}" "${progress_field}")"
    status_text="$(colorize_status "${status}" "${status_field}")"

    printf "%-4s  %s  %s  %s\n" "${plan_id}" "${status_text}" "${progress_text}" "${title}"

    if [[ "${status}" != "Draft" && "${status}" != "Active" && "${status}" != "Paused" && "${status}" != "Done" && "${status}" != "Cancelled" ]]; then
        printf "  %sinvalid status%s in %s: %s\n" "${color_red}" "${color_reset}" "${filename}" "${status}" >&2
        failures=$((failures + 1))
    fi

    if [[ "${total_count}" -gt 0 ]]; then
        if [[ "${complete_count}" -eq "${total_count}" && "${status}" != "Done" && "${status}" != "Cancelled" ]]; then
            printf "  %sexpected Done or Cancelled%s in %s: all checklist items are done\n" "${color_red}" "${color_reset}" "${filename}" >&2
            failures=$((failures + 1))
        fi

        if [[ "${complete_count}" -lt "${total_count}" && "${status}" != "Draft" && "${status}" != "Active" && "${status}" != "Paused" && "${status}" != "Cancelled" ]]; then
            printf "  %sexpected Draft, Active, Paused, or Cancelled%s in %s: checklist still has open items\n" "${color_red}" "${color_reset}" "${filename}" >&2
            failures=$((failures + 1))
        fi
    fi
done

if [[ "${found}" -eq 0 ]]; then
    printf "%sno plan files found%s in %s\n" "${color_red}" "${color_reset}" "${plans_dir}" >&2
    exit 1
fi

if [[ "${failures}" -gt 0 ]]; then
    echo
    printf "%splan status check failed%s with %s issue(s).\n" "${color_red}" "${color_reset}" "${failures}" >&2
    exit 1
fi

echo
printf "%splan status check passed.%s\n" "${color_green}" "${color_reset}"
