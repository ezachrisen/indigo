# Use Cases
This document describes the ways you can use Indigo to solve various use cases. A combination of a rule hierarchy and evaluation options give you flexibility beyond evaluating an expression. 



## Case 1: Is this rule true?
The most basic use is to check whether a single rule is satisfied by the input provided. The structure to do this looks like this:


### Rules

```
 
  [*] Rule "myID"
	  - Expression "x>100"
```


### Execution
In pseudocode, to evaluate ``data`` against "myRule", you would execute this:

``` go
r, err := indigo.Evaluate(rule, data)
```

### Results
The true/false result is in r.Pass


## Case 2: Which of these rules are true?
This use case lends itself to situations where you need to determine which, if any, rules match the input.

### Examples
- Which offers (rules) does a customer (input) qualify for?
- Which alarms (rules) should we raise for the server metrics (data)?


### Rules
The rule structure looks like this for the alarms example:

```
  [*] Rule "alarms"
      Options: 
        StopFirstPositiveChild: false 
        StopFirstNegativeChild: false 
        ReturnPass:             true 
        ReturnFail:             false
      Child rules:
        [*] Rule "CPUHigh"
            - Expression: "cpu.utilization > 90"
        [*] Rule "DiskLow"
            - Expression: "disk.freespace < 70"
        [*] Rule "MemoryHigh"
            - Expression: "memory.utilization > 90"
```


For this use case it's important to specify that you want all true rules returned (``ReturnPass: true``) and not to stop if you encounter any false rules (``StopFirstNegativeChild: false``).

### Execution
You specify that you want to evaluate the "alarms" rule, which in fact doesn't have a rule expression, but does have child rules. 

``` go
r, err := indigo.Evaluate(alarms_rule, data)
```


## Case 3: Is at least 1 rule true? 
With this method there may be many rules to evaluate, but we are only interested in if one of the rules is true. The outcome we are interested is binary: yes or no. 

This is typically the case in role-based access scenarios, where a user may qualify for access to a resource either by being the owner of the resource, by membership in a role, or some other reason. 


### Examples
- Can the user perform the requested action (role-based access)?

### Rules
By specifying that we want to stop at the first positive child, we will return the first rule that passes, but not evaluate the rest of the rules.

```
  [*] Rule "access"
      Options: 
        StopFirstPositiveChild: true 
        StopFirstNegativeChild: false 
        ReturnPass:             true
        ReturnFail:             false
      Child rules:
        [*] Rule "Owner"
            - Expression: "resource.owner == user.user_id"
        [*] Rule "Admin"
            - Expression: "'admin' in user.roles"
        [*] Rule "GroupOwner"
            - Expression: "resource.group_id in user.group_ids"
```


### Execution

``` go
r, err := indigo.Evaluate(access_rule, data)
```

### Results
The first rule that passed is available in ``r.Results``.


## Case 4: Condition inheritance
For some situations it's useful to be able to have a complex condition be evaluated once and its result be inherited by the child rules. By using a combination of rules, child rules and the execution options, we can use the parent rule as a "gate" to access the child rules. 

The same can be achieved with an AND condition in a child rule, but the inheritance technique can be used to avoid evaluating potentially thousands of rules if the parent condition is false. 

### Examples
- For all teams that have a winning record, identify the players who have a batting average > .250
- Perform deep fraud checks for transactions over $10,000


### Rules
In this example we're going to use an expression and the ``StopIfParentNegative`` option on the parent rule to control evaluation of the child rules. 

```
  [*] Rule "winners_high_avg"
      Expression: "team.winning_pct > 0.50"
      Options: 
        StopIfParentNegative:   true
        StopFirstPositiveChild: false
        StopFirstNegativeChild: false 
        ReturnPass:             true 
        ReturnFail:             false
      Child rules:
        [*] Rule "BattingAvg"
            - Expression: "player.batting_avg > 0.250"

```

### Execution

``` go
r, err := indigo.Evaluate(winners_high_avg_rule, data)
```


### Results
The players from teams with winning records are in ``r.Results`. 


## Case 5: Why was the request denied? 
When a rule expression is evaluated, it is either true or false, and the reason why (which predicate was false) is not immediately obvious. If you enable diagnostics you can get this information from CEL, but diagnostics are very performance expensive. You could also achieve a similar effect with conditional object construction (see  cel/examples_test.go), but that also has a performance penalty. 

Instead, break the rule into child rules, and attach an explanation to each of the child rules. 

## Examples
- A student was denied enrollment in a class - why?

### Rules
In this case we have an expression on the parent rule, and the same expression broken up into child rules. (In a future enhancement, the result of the child rules will roll up to the parent rule, obviating the need to repeat the expression.)

Note that we will evaluate the child rules if the parent is negative, so that we can get an explanation. We will also return only the child rules that fail. 

```
  [*] Rule "enrollment_qualification"
      Expression: "student.major == 'Business' && student.bursar_status != 'Hold' && class.current_enrollment < 30"
      Options: 
        StopIfParentNegative:   false
        StopFirstPositiveChild: false
        StopFirstNegativeChild: false 
        ReturnPass:             false
        ReturnFail:             true        <<< Indicates we want to see the rules that were NOT satsified
      Child rules:
        [*] Rule "major"
            - Expression: "student.major == 'Business'"
            - Meta: string:"Student's major must be Business"
        [*] Rule "bursar"
            - Expression: "student.bursar_status != 'Hold'"
            - Meta: string:"Student cannot have a financial hold from the Bursar's office"
        [*] Rule "bursar"
            - Expression: "class.current_enrollment < 30"
            - Meta: string:"Max class enrollment met"
```

### Execution

``` go
r, err := indigo.Evaluate(enrollment_qualification_rule, data)
```


### Results
Whether the enrollment was successful will be in the parent result (``r.Pass``). The reasons why not will be in ``r.Results``. The calling application must inspect the Meta field of each result to get the user-friendly message. 



