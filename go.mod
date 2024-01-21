module github.com/venkatsvpr/golang-lru/v2

go 1.18

replace github.com/hashicorp/golang-lru/v2 => ./

replace github.com/hashicorp/golang-lru/v2/internal => ./internal

require github.com/hashicorp/golang-lru/v2 v2.0.0-00010101000000-000000000000
