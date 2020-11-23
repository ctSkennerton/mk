% MK(1) Mk user manual
% Daniel C. Jones, Connor T. Skennerton
% November 16, 2020

# NAME
mk - maintain (make) related files

# SYNOPSIS
`mk [ -f mkfile ] ...  [ option ... ] [ target ... ]`


# DESCRIPTION
`Mk` uses the dependency rules specified in mkfile to control
the update (usually by compilation) of targets (usually
files) from the source files upon which they depend.  The
mkfile (default `mkfile`) contains a rule for each target
that identifies the files and other targets upon which it
depends and a script or recipe, to update the target.
The script is run if the target does not exist or if it is
older than any of the files it depends on.  Mkfile may also
contain meta-rules that define actions for updating implicit
targets.  If no target is specified, the target of the first
rule (not meta-rule) in mkfile is updated.


Options are:

-f
:   use the given file as mkfile. default is *mkfile*

-n
:   print commands without actually executing

-r
:   force building of just targets

-a
:   force building of all dependencies

-p
:   maximum number of jobs to execute in parallel. Default is the number of CPUs

-i
:   prompt before executing rules

-q
:   don't print recipes before executing them


## The mkfile

A mkfile consists of assignments (described under `Environment')
 and rules. A rule contains targets and a tail. A target
  is a literal string and is normally a file name.  The
tail contains zero or more prerequisites and an optional
recipe.  Each line of the recipe must
begin with white space.  A rule takes the form

    target: prereq1 prereq2
            recipe using prereq1, prereq2 to build target


After the colon on the target line, a rule may specify
attributes, described below.

A meta-rule has a target of the form A%B where A and B are
(possibly empty) strings.  A meta-rule acts as a rule for
any potential target whose name matches A%B with % replaced
by an arbitrary string, called the stem. In interpreting a
meta-rule, the stem is substituted for all occurrences of %
in the prerequisite names.  In the recipe of a meta-rule,
the environment variable `$stem` contains the string matched
by the %.  For example, a meta-rule to compile a C program
using might be:

    %: %.c
        cc -o $stem $stem.c


The text of the mkfile is processed as follows.  Lines
beginning with `<` followed by a file name are replaced by the
contents of the named file.  Lines beginning with `<|` followed
 by a file name are replaced by the output of the exe-
cution of the named file.  Blank lines and comments, which
run from unquoted `#` characters to the following newline, are
deleted.  The character sequence backslash-newline is
deleted, so long lines in mkfile may be folded.  Non-recipe
lines are processed by substituting for `{command}` the output
of the command when run by rc. References to variables
are replaced by the variables' values.

Assignments and rules are distinguished by the first
unquoted occurrence of `:` (rule) or `=` (assignment).

A later rule may modify or override an existing rule under
the following conditions:

-   If the targets of the rules exactly match and one rule
    contains only a prerequisite clause and no recipe, the
    clause is added to the prerequisites of the other rule.
    If either or both targets are virtual, the recipe is
    always executed.
-   If the targets of the rules match exactly and the 
    prerequisites do not match and both rules contain recipes,
    `mk` reports an "ambiguous recipe" error.
-   If the target and prerequisites of both rules match
    exactly, the second rule overrides the first.

### Environment
Rules may make use of environment variables.  A legal
reference of the form `$OBJ` is expanded. A reference
of the form `${name:A%B=C%D}`, where A, B, C, D are (possibly
empty) strings, has the value formed by expanding
`$name` and substituting C for A and D for B in each word in
`$name` that matches pattern A%B.

Variables can be set by assignments of the form

    var=[attr=]value

Blanks in the value break it into words, but without
the surrounding parentheses.  Such variables are
exported to the environment of recipes as they are executed,
unless U, the only legal attribute attr, is present.  The
initial value of a variable is taken from (in increasing
order of precedence) the default values below, mk's environment,
the mkfiles, and any command line assignment as an
argument to mk. A variable assignment argument overrides the
first (but not any subsequent) assignment to that variable.

The variable MKFLAGS contains all the option arguments
(arguments starting with '-' or containing '=') and MKARGS
contains all the targets in the call to mk.

### Execution
During execution, mk determines which targets must be
updated, and in what order, to build the names specified on
the command line.  It then runs the associated recipes.

A target is considered up to date if it has no prerequisites
or if all its prerequisites are up to date and it is newer
than all its prerequisites.  Once the recipe for a target
has executed, the target is considered up to date.

The date stamp used to determine if a target is up to date
is computed differently for different types of targets.  If
a target is virtual (the target of a rule with the V
attribute), its date stamp is initially zero; when the target
is updated the date stamp is set to the most recent date
stamp of its prerequisites.  Otherwise, if a target does not
exist as a file, its date stamp is set to the most recent
date stamp of its prerequisites, or zero if it has no prerequisites.
For URLs the `Last-Modified` header returned from a HTTP HEAD request
is used to determine if the target is up to date.
Otherwise, the target is the name of a file and
the target's date stamp is always that file's modification
date.  The date stamp is computed when the target is needed
in the execution of a rule; it is not a static value.

Nonexistent targets that have prerequisites and are themselves
prerequisites are treated specially.  Such a target `t`
is given the date stamp of its most recent prerequisite and
if this causes all the targets which have `t` as a prerequisite
to be up to date, `t` is considered up to date.  Otherwise, 
`t` is made in the normal fashion.  

Files may be made in any order that respects the preceding
restrictions.

A recipe is executed by supplying the recipe as standard
input to the command, `sh`, unless The `S` attribute is set,
which defines an alternative program to run the recipe

The environment is augmented by the following variables:

$alltarget    
:   all the targets of this rule.

$newprereq    
:   the prerequisites that caused this rule to execute.

$newmember    
:   the prerequisites that are members of an
    aggregate that caused this rule to execute.
    When the prerequisites of a rule are members
    of an aggregate, $newprereq contains the name
    of the aggregate and out of date members,
    while $newmember contains only the name of the
    members.

$nproc        
:   the process slot for this recipe.  It satisfies 0≤$nproc<$NPROC.

$pid          
:   the process id for the mk executing the recipe.

$prereq       
:   all the prerequisites for this rule.

$stem         
:   if this is a meta-rule, $stem is the string
    that matched % or &.  Otherwise, it is empty.
    For regular expression meta-rules (see below),
    the variables `stem0, ..., stem9` are set to
    the corresponding subexpressions.

$target       
:   the targets for this rule that need to be remade.

These variables are available only during the execution of a
recipe, not while evaluating the mkfile.

Unless the rule has the Q attribute, the recipe is printed
prior to execution with recognizable environment variables
expanded.  Commands returning nonempty status
cause `mk` to terminate.

Recipes and backquoted commands in places such as assignments 
execute in a copy of mk's environment; changes they
make to environment variables are not visible from mk.

Variable substitution in a rule is done when the rule is
read; variable substitution in the recipe is done when the
recipe is executed.  For example:

    bar = a.c
    foo: $bar
            $CC -o foo $bar
    bar = b.c

will compile b.c into foo, if a.c is newer than foo.

### Aggregates
Names of the form a(b) refer to member b of the aggregate a.
Currently, the only aggregates supported are ar(1) archives.

### Attributes
The colon separating the target from the prerequisites may
be immediately followed by attributes and another colon.
The attributes are:

D    
:   If the recipe exits with a non-null status, the target
    is deleted.

E    
:   Continue execution if the recipe draws errors.

N    
:   If there is no recipe, the target has its time updated.

n    
:   The rule is a meta-rule that cannot be a target of a
    virtual rule.  Only files match the pattern in the
    target.

P    
:   The characters after the P until the terminating : are
    taken as a program name.  It will be invoked as rc -c
    prog 'arg1' 'arg2' and should return a null exit status
    if and only if arg1 is up to date with respect to arg2.
    Date stamps are still propagated in the normal way.
    This attribute is not compatible with the S attribute.

S
:   Characters after S until the terminating : are taken as
    a program name. This program will be used to execute the
    recipe. This attrbiute is not compatible with the P attribute.

Q    
:   The recipe is not printed prior to execution.

R    
:   The rule is a meta-rule using regular expressions.  In
    the rule, % has no special meaning.  The target is
    interpreted as a regular expression as defined in
    regexp(6). The prerequisites may contain references to
    subexpressions in form \n.

U    
:   The targets are considered to have been updated even if
    the recipe did not do so.

V    
:   The targets of this rule are marked as virtual.  They
    are distinct from files of the same name.

# EXAMPLES
A simple mkfile to compile a program:

    </$objtype/mkfile

    prog:   a.$O b.$O c.$O
            $LD $LDFLAGS -o $target $prereq

    %.$O:   %.c
            $CC $CFLAGS $stem.c

Override flag settings in the mkfile:

    % mk target 'CFLAGS=-S -w'

Maintain a library:

    libc.a(%.$O):N: %.$O
    libc.a: libc.a(abs.$O) libc.a(access.$O) libc.a(alarm.$O) ...
            ar r libc.a $newmember

String expression variables to derive names from a master
list:

    NAMES=alloc arc bquote builtins expand main match mk var word
    OBJ=${NAMES:%=%.$O}

Regular expression meta-rules:

    ([^/]*)/(.*)\.$O:R:  $stem1/$stem2.c
            cd $stem1; $CC $CFLAGS $stem2.c

A correct way to deal with yacc(1) grammars.  The file lex.c
includes the file x.tab.h rather than y.tab.h in order to
reflect changes in content, not just modification time.

    lex.$O: x.tab.h
    x.tab.h:        y.tab.h
            cmp -s x.tab.h y.tab.h || cp y.tab.h x.tab.h
    y.tab.c y.tab.h:        gram.y
            $YACC -d gram.y

The above example could also use the P attribute for the
x.tab.h rule:

    x.tab.h:Pcmp -s:        y.tab.h
            cp y.tab.h x.tab.h


# SEE ALSO
A. Hume, "Mk: a Successor to Make".

Andrew G. Hume and Bob Flandrena, "Maintaining Files on
Plan 9 with Mk".

Most of the content of this manual is copied from the 
Plan 9 mk manual available here:
http://man.cat-v.org/plan_9/1/mk

# BUGS