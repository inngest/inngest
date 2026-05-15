frand
-----

[![GoDoc](https://godoc.org/lukechampine.com/frand?status.svg)](https://godoc.org/lukechampine.com/frand)
[![Go Report Card](http://goreportcard.com/badge/lukechampine.com/frand)](https://goreportcard.com/report/lukechampine.com/frand)

```
go get lukechampine.com/frand
```

`frand` is a fast CSPRNG in userspace, implemented as a lightweight wrapper
around the [`(math/rand/v2).ChaCha8`](https://go.dev/src/math/rand/v2/chacha8.go) generator. The initial
cipher key is derived from the kernel CSPRNG, after which [no new entropy is
ever mixed in](https://blog.cr.yp.to/20140205-entropy.html).

The primary advantage of `frand` over `crypto/rand` is speed: when generating
large amounts of random data, `frand` is 20x faster than `crypto/rand`, and over
100x faster when generating data in parallel. `frand` is also far more
convenient to use than `crypto/rand`, as its methods cannot return errors, and
helper methods such as `Intn` and `Perm` are provided. In short, it is as fast
and convenient as `math/rand`, but far more secure.

## Design

`frand` was previously implemented as a [FKE-CSPRNG](https://blog.cr.yp.to/20170723-random.html).
Since the addition of `ChaCha8` in Go 1.23, this is no longer necessary, as the
stdlib generator is equally fast and secure. If you're curious about the old
implementation, you can read about it [here](https://github.com/lukechampine/frand/blob/3f951cc7c03d029aba12f89106666489fd8c769b/README.md).

Calls to global functions like `Read` and `Intn` are serviced by a `sync.Pool`
of generators derived from the master generator. This allows `frand` to scale
near-linearly across CPU cores, which is what makes it so much faster than
`crypto/rand` and `math/rand` in parallel benchmarks.

## Security

This is a slightly hot take, but I do not believe there is a meaningful
difference in security between using `crypto/rand` and a `crypto/rand`-seeded
ChaCha8 generator. There are caveats, of course: for example, since the
generator is in userspace, it is somewhat easier to access its internal state
and predict future. But if an attacker can access the internal state of your
program, surely you have bigger fish to fry. If `crypto/rand` makes you feel
safer, great, use it. But I don't think it's actually any more secure.

Even if you aren't comfortable using `frand` for critical secrets, you should
still use it everywhere you would normally use an insecure PRNG like
`math/rand`. Go programmers frequently use `math/rand` because it is faster and
more convenient than `crypto/rand`, and only use `crypto/rand` when "true"
randomness is required. This is an unfortunate state of affairs, because misuse
of `math/rand` can lead to bugs or vulnerabilities. For example, using
`math/rand` to generate "random" test cases will produce identical coverage from
run-to-run if the generator is left unseeded. More seriously, using `math/rand`
to salt a hash table may make your program [vulnerable to denial of service attacks](https://stackoverflow.com/questions/52184366/why-does-hashmap-need-a-cryptographically-secure-hashing-function).
Substituting `frand` for `math/rand` would avoid both of these outcomes.
