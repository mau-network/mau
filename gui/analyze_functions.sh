#!/bin/bash
# Analyze function lengths in Go files

for file in *.go; do
    if [[ "$file" == *_test.go ]]; then
        continue
    fi
    
    echo "=== $file ==="
    awk '
    /^func / {
        fname = $0
        line_count = 0
        start = NR
        in_func = 1
    }
    in_func {
        line_count++
        if ($0 ~ /^}$/ && line_count > 1) {
            if (line_count > 20) {
                printf "%3d lines (L%d-%d): %s\n", line_count, start, NR, fname
            }
            in_func = 0
        }
    }
    ' "$file"
done
