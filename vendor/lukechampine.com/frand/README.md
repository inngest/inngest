frand
-----

[![GoDoc](https://godoc.org/lukechampine.com/frand?status.svg)](https://godoc.org/lukechampine.com/frand)
[![Go Report Card](http://goreportcard.com/badge/lukechampine.com/frand)](https://goreportcard.com/report/lukechampine.com/frand)

```
go get lukechampine.com/frand
```

`frand` is a [fast-key-erasure
CSPRNG](https://blog.cr.yp.to/20170723-random.html) in userspace. Its output is
sourced from the keystream of a [ChaCha](https://en.wikipedia.org/wiki/Salsa20#ChaCha_variant)
cipher, much like the CSPRNG in the Linux and BSD kernels. The initial cipher key is
derived from the kernel CSPRNG, after which [no new entropy is ever mixed in](https://blog.cr.yp.to/20140205-entropy.html).

The primary advantage of `frand` over `crypto/rand` is speed: when generating
large amounts of random data, `frand` is 20x faster than `crypto/rand`, and over
100x faster when generating data in parallel. `frand` is also far more
convenient to use than `crypto/rand`, as its methods cannot return errors, and
helper methods such as `Intn` and `Perm` are provided. In short, it is as fast
and convenient as `math/rand`, but far more secure.

## Design

`frand` closely follows the FKE-CSPRNG design linked above. The generator
maintains a buffer consisting of a ChaCha key followed by random data. When a
caller requests data, the generator fills the request from its buffer, and
immediately overwrites that portion of the buffer with zeros. If the buffer is
exhausted, the generator "refills" the buffer by xoring it with the ChaCha key's
keystream; thus, the key is only used once, and immediately overwrites itself.
This provides [forward secrecy](https://en.wikipedia.org/wiki/Forward_secrecy):
even if an attacker learns the secret key, they cannot learn what data was
previously output from the generator.

One optimization is implemented. If the caller requests a large amount of data
(larger than the generator's buffer), the generator does not repeatedly fill and
drain its buffer. Instead, it requests 32 bytes from itself, and uses this as a
ChaCha key to directly generate the amount of data requested. The key is then
overwritten with zeros.

The default generator uses ChaCha12. ChaCha8 is significantly weaker without
being much faster; ChaCha20 is significantly slower without being much stronger.

At `init` time, a "master" generator is created using an initial key sourced
from `crypto/rand`. New generators source their initial keys from this master
generator. Following [djb's recommendation](https://blog.cr.yp.to/20140205-entropy.html),
the generator never mixes in new entropy.

Calls to global functions like `Read` and `Intn` are serviced by a `sync.Pool`
of generators derived from the master generator. This allows `frand` to scale
near-linearly across CPU cores, which is what makes it so much faster than
`crypto/rand` and `math/rand` in parallel benchmarks.


## Security

Some may object to substituting a userspace CSPRNG for the kernel's
CSPRNG. Certain userspace CSPRNGs (e.g. [OpenSSL](https://research.swtch.com/openssl))
have contained bugs, causing serious vulnerabilities. `frand` has some
advantages over these CSPRNGs: it never mixes in new entropy, it doesn't
maintain an "entropy estimate," and its implementation is extremely simple.
Still, if you only need to generate a handful of secret keys, you should
probably stick with `crypto/rand`.

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


## Benchmarks


| Benchmark                | `frand`        |`crypto/rand`  | `math/rand` (insecure) | [`fastrand`](https://gitlab.com/NebulousLabs/fastrand) |
|--------------------------|----------------|---------------|------------------------|------------|
| Read (32b)               | **964 MB/s**   | 59 MB/s       | 634.21 MB/s            | 215 MB/s   |
| Read (32b, concurrent)   | **3566 MB/s**  | 70 MB/s       | 198.97 MB/s            | 615 MB/s   |
| Read (512kb)             | **5094 MB/s**  | 239 MB/s      | 965.85 MB/s            | 452 MB/s   |
| Read (512kb, concurrent) | **19665 MB/s** | 191 MB/s      | 958.01 MB/s            | 1599 MB/s  |
| Intn (n =~ 4e18)         | 45 ns/op       | 725 ns/op     | **20 ns/op**           | 210 ns/op  |
| BigIntn (n = 2^630)      | **223 ns/op**  | 1013 ns/op    | 295 ns/op              | 468 ns/op  |
| Perm (n = 32)            | 954 ns/op      | 17197 ns/op   | **789 ns/op**          | 5021 ns/op |

Benchmark details:

"Concurrent" means the `Read` function was called in parallel from 64 goroutines.

`Intn` was benchmarked with n = 2^62 + 1, which maximizes the number of expected
retries required to remove bias. The number of expected retries is 1.333.
