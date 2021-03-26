// Package Indigo provides a rules engine process that uses an instance of the
// Evaluator interface to perform evaluation of rules.
//
// Indigo itself does not specify a language for rules, relying instead on the Evaluator's
// rule language.
//
// Typical use is as follows:
//
//  1. Declare a schema describing the type of data you will be processing
//  2. Create a rule, possibly with many child rules
//  3. Create an engine
//  4. Use the engine to compile the rule
//  5. Use the engine to evaluate the rule against a set of input data
//  6. Inspect the results
//
// Depending on your need, many options are available to control how rules are evaluated, and
// how results are returned.
//
// Rule Ownership and Modification
//
// The calling application is responsible for managing the lifecycle of rules, including ensuring
// concurrency safety. Specifically, this means:
//  1. You must not allow changes to a rule during compilation.
//  2. You may not modify the rule after compilation and before evaluation.
//  3. You must not allow changes to a rule during evaluation.
//  4. A rule must not be a child rule of more than one parent.
//
// Breaking these rules could lead to race conditions or unexpected outcomes.
//
// The simplest and safest way to use the rules engine, is of course to load the rules at startup and
// never change them. That may be appropriate for some use cases, but in many cases the rules are continually
// being changed by humans or other processes.
//
// If you want to modify a rule, for example to remove a child rule, you can make the modification
// then compile it again, before evaluating it. During these steps you must again make sure no other
// process is using the rule for evaluation or is updating it.
//
// Structuring Rule Hierarchies for Updates
//
// The ability to organize rules in a hierarchy is useful to ensure that rule updates are atomic and consistent.
//
// You should structure the hierarchy so that a rule and its children can be seend as a
// "transaction" as far as updates are concerned.
//
// In this example, where Indigo is being used to enforce firewall rules, being able
// to update ALL firewall rules as a group, rather than one by one (where one update may fail)
// is important.
//
//   Firewall Rules (parent)
//     "Deny all traffic" (child 1)
//     "Allow traffic from known_IPs" (child 2)
//
// If the user changes child 1 to be "Allow all traffic" and changes child 2 to "Deny all traffic, except for known_IPs",
// there's a risk that child 1 is changed first, without the child 2 change happening. This would leave us with this:
//
//   Firewall Rules (parent)
//     "Allow all traffic" (child 1)
//     "Allow traffic from known_IPs" (child 2)
//
// This is clearly bad!
//
// Instead of accepting a change to child 1 and child 2 separately, ONLY accept a change to your rule hierarchy for the
// Firewall Rules parent. That way the update succeeds or fails as a "transaction".
//
// If Firewall Rules is itself a child of a larger set of parent rules, it's recommended to compile the Firewall Rules parent and
// children BEFORE adding it to its eventual parent. That way you ensure that if compilation of Firewall Rules fails, the
// "production" firewall rules are still intact.
//
package indigo
