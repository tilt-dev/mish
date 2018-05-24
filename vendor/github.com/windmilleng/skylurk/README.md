# skylurk

A shadow of the Skylark interpreter.

### Description

Windmill discovers jobs by running a Skylark interpreter in partial-evaluation mode.

In this mode, we have two special values:

- `waitingValue`: We have scheduled this job, but don't want to wait for it right now

- `failValue`: This job failed, and we do not expect its output to be well-formed

We propagate these two values throughout the program, such that the result of
any operator is a failValue if one of its operands is a failValue, and is a
waitingValue if one of its operands is a waitingValue.

This is a very different approach than Bazel takes. In the Bazel build language,
the language itself separates execution into two phases: declaring each job and
its dependencies, then executing the job graph. To do this, the language
declares rules by name, and those rules depend on other rules by name.

In Windmill, the job graph is dynamic. One job target could shard out into
hundreds of other job targets. The evaluator tries to "explore" this graph
without building it all upfront.

To do this, we modify the Skylark interpreter to behave more like a partial
evaluator, with better support for dynamic values that represent multiple
possible states. Note that we use the Skylark syntax and value data structures
from upstream. We've only forked the evaluation engine.

We believe this is less sound than the Bazel approach. But we hope it will
provide a better developer experience. We think this kind of developer
experience risk is fundamental to Windmill and worth taking.  If it doesn't, we
can move towards a more declarative approach like Bazel.

### Legal

Skylark in Go is Copyright (c) 2017 The Bazel Authors.
All rights reserved.

It is provided under a 3-clause BSD license:
[LICENSE](https://github.com/google/skylark/blob/master/LICENSE).

The name "Skylark" is a code name of the Bazel project.
We plan to rename the language before the end of 2017 to reflect its
applicability to projects unrelated to Bazel.

Skylark in Go is not an official Google product.
