---
title: "Frequently Asked Questions"
---


{{% faq/list %}}
{{% faq/question id="false-positives" question="Staticcheck is wrong, what should I do?" %}}
First, make sure that Staticcheck is actually wrong.
It can find very subtle bugs, and what may look like a false positive at first glance is usually a genuine bug.
There is a long list of competent programmers
[who got it wrong before.](https://github.com/dominikh/go-tools/issues?q=is%3Aissue+label%3Afalse-positive+label%3Ainvalid+is%3Aclosed)

However, sometimes Staticcheck _is_ wrong and you want to suppress a warning to get on with your work.
In that case, you can use [ignore directives to ignore specific problems]({{< relref "/docs/configuration/#ignoring-problems" >}}).
You should also [report the false positive](https://github.com/dominikh/go-tools/issues/new?assignees=&labels=false-positive%2C+needs-triage&template=1_false_positive.md&title=) so that we can fix it.
We don't expect users to have to ignore many problems, and we always aim to avoid false positives.

Some checks, particularly those in the `ST` (stylecheck) category, may not be applicable to your code base at all. In that case, you should disable the check using the
[`checks` option]({{< relref "/docs/configuration/options#checks" >}})
in your [configuration]({{< relref "/docs/configuration/#configuration-files" >}}).
{{% /faq/question %}}
