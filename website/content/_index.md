---
title: Staticcheck
---


{{< blocks/lead type="section" color="primary" >}}
<img style="height: 300px" src="/img/logo.svg" alt="Our mascot, the engineer" />

Staticcheck is a state of the art linter for the Go programming language.
Using static analysis, it finds bugs and performance issues, offers simplifications, and enforces style rules.


<div class="mx-auto">
	<a class="btn btn-lg btn-secondary mr-3 mb-4" href="{{< relref "/docs" >}}">
		Get started <i class="fas fa-arrow-alt-circle-right ml-2"></i>
	</a>
</div>
{{< /blocks/lead >}}

{{< blocks/section color="white" >}}

{{% blocks/feature title="Correctness" icon="fa-check" %}}
Code obviously has to be correct. Unfortunately, it never
is. Some code is more correct than other, but there'll
always be bugs. Tests catch some, your peers catch others,
and Staticcheck catches some more.
{{% /blocks/feature %}}


{{% blocks/feature title="Simplicity" icon="fa-circle" %}}
After correctness comes simplicity. There are many ways to
skin a cat (but please don't), but some are unnecessarily
elaborate. Staticcheck helps you replace complex code with
simple code.
{{% /blocks/feature %}}

{{% blocks/feature title="Maintainability" icon="fa-screwdriver" %}}
Code is a living thing. If it's not maintained regularly,
it will become stale and unreliable. It won't catch up
when its dependencies move on or guidelines change.
Staticcheck is like a sitter for your code for when you
don't have time.
{{% /blocks/feature %}}


{{% blocks/feature title="Exhaustiveness" icon="fas fa-tasks" url="/docs/checks" %}}
More than 100 checks ensure that your code is in tip-top shape.
{{% /blocks/feature %}}


{{% blocks/feature title="Integration" icon="fab fa-github"  %}}
Staticcheck can easily be integrated with your code review and CI systems, preventing buggy code from ever getting committed.
{{% /blocks/feature %}}


{{% blocks/feature title="Editor integration" icon="fa-i-cursor" url="https://github.com/golang/tools/blob/master/gopls/doc/settings.md#staticcheck-bool" %}}
Staticcheck is part of most major Go editors and has been integrated with gopls, finding bugs and offering automatic fixes.
{{% /blocks/feature %}}


{{< /blocks/section >}}

<link rel="prefetch" href="/docs/">
