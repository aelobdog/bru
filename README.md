# Bru

## Introduction
Welcome to Bru's official documentation. If you have been following the Youtube
series, you will find this document to be completely compatible with it. If you
haven't been following it, no problems! This document is complete in itself,
you'll do just fine even if you are new to Bru. This document takes you through
the whole process, from writing your first bru file, to silumating it.

Interested? Cool! You've come to the right place.

## Prerequisites
There aren't that many prerequisites to Bru. In terms of software, you will
need to install the Go programming language first. This can be done easily for
any platform, be it Windows, Mac or Gnu/Linux. You can find the instructions
for installing Go for your system at the link below.

[Go Programming Language](https://golang.org/dl/)

The other software prerequisite for Bru is the Bru source file itself. This can
be obtained from my GitHub repository linked below. 

[Bru GitHub repository](https://github.com/ashvin-godbole/bru)

You can download the file in any way you like. If you have Git installed on 
your computer, you can clone the repository using the command given below.

Clone the GitHub repository using : ```git clone https://github.com/ashvin-godbole/bru```

Once you have installed Go and downloaded the Bru source code, you're all set.
Now the only other thing that you need is an interest in goofing around with
tool!

## Bru Project Structure
Every Bru project consists of HDL files and their corresponding SCRIPT files.
The HDL files describe the construction of a particular circuit and the SCRIPT
files contain the inputs to be provided to the circuits described. These files
don't require any specific file extensions. This means that you can give these
HDL and SCRIPT files may or may not have any file extension and if they do have
an extension, it may be anything you wish.

One thing you need to keep in mind is that all files in your project must be
stored in ONE FOLDER ONLY. This means that you can't, at the moment, segregate
your HDL and SCRIPT files in their own separate folders. This is a bug that we 
must unfortunately must live with, for now at least.

So your folder must look something like this: 

```
- Project Root
    - HDL_FILE_ONE
    - HDL_FILE_TWO
    - SCRIPT_FILE_ONE
    - SCRIPT_FILE_TWO
    - ...
```

## Bru Hardware Description Language
Like many other tools, Bru uses a flavor of HDL to describe the structure of
a circuit. This falvor of HDL is designed to be super simple. There are only 6
keywords in total in Bru's HDL. They are : *, IN, OUT, CON, END, SIM, CLK. (yes
, the '*' character is a keyword).

Let's look at a sample HDL file:
```
* nand
IN  i1 i2
OUT o1
CON
    t1 = and(i1, i2)
    o1 = not(t1)
END
```
(Note: Indentation isn't compulsory, it may be ommited as per your preferences)

This is a simple "nand" gate in BruHDl. If you have any experience with any
of the other HDLs out there, you may find certain similarities and a LOT of
differences when you compare BruHDL with them. Most of the syntax choices
that have been made for BruHDL have not been chosen for any specific reason,
other than that I felt like implementing it in a certain way to either make
it simpler for you to write in it or to make it simpler for me to write its
implementation.

### Syntax
Wondering what you just read (above)? No worries! here's an explanation for it.
The syntax of Bru's HDL is divided into the following sections, roughly:

* Component Declaration
* Special Flags/Keywords
* The Input and Output specifiers.
* Component design instructions
    
Lets look at each one of these one by one.

#### Component Declaration.
Every component starts with a '*' followed by the name of the component.
The name of any component may not have spaces in it. If you want to separate
words in the name, you may use underscores(_) or pascalCase, but not spaces.
This section should also always be followed by the components 'definition'
or its 'body'. From the above example, 'nand' may be declared as :
```
* nand
```

(Note: the space between '*' and the name is just for clarity. You may have as 
many spaces as you want, or even none, after the '*')


#### Special Flags/Keywords
Bru provides 2 flags/keywords for which tell the simulator different things. 
These flags are :
```
- SIM
- CLK
```

If you are designing a component, chances are you want to also simulate it
to check its 'correctness'. Bru allows you to simulate _one component_ per
'run'. To indicate which one of the multiple components you may have in a
file, you can use the SIM flag. This tag would immediately follow the
declaration of the component like so :
```
* nand
SIM
```

If you want to indicate to Bru that a particular component is sequential, 
not combinational in nature, you may use the CLK flag. More details on this
flag later. For now, just know that there is another flag. When you use it, 
it will look like this :
```
* some_sequential_circuit
CLK
```

What if you want to simulate a sequential component ? Well it's simple.
Just stack the flags after the declaration like so :
```
* some_sequential_circuit
SIM
CLK
```
(Note: The order of flags in not important)

#### The Input and Output specifiers.
Following the optional flags are the INput and OUTput lines. These are very
simple to understand, other than maybe one case, where you may want one of the
circuit's outputs to link back into one of its inputs. More on that in a bit.

For any component, all its inputs may be specified as a list of SPACE
separated indentifiers. This list must follow the IN keyword, where 'IN'
has to be in _UPPERCASE_. For example, for the 'nand' gate we would have
(assuming that the nand gate is a 2 input gate)
```
IN i1 i2
```
(Note: using 'i' followed by a number is a convention that I follow. You may
use any names for the inputs that you wish, like a, b, inp1, inp2 etc.)

You may also use input buffers if the number of inputs is large. This means
that if you want to represent 2 inputs as a buffer, you may do so like this :
```
IN i[2]
```
and the individual elements of this can be used as you would access the
elements of a zero-indexed array, like so:
```
i[0], i[1]
```

All of the aforementioned things can be done the _exact same way for outputs_
using the OUT keyword.

An interesting, but weird case arises if you want to connect the output of
a particular component back into one of its inputs. This may be useful when
designing sequential circuits like memory elements and such. Bru supports
this feature too ! If you want to link any output back to any of the inputs
you can do this :
```
IN i1 (i2|o1)
OUT o1 o2
```

(Note: the position of (i2|o1) is not fixed. It may occur anywhere within the 
inputs list)

(Note: This feature currently works only on components that are being simulated.
I cannot guarentee that it will work in any other case. This feature is not 
meant to be used very often and is untested. Please think before using this)


#### Component design instructions
This is the last section of a component's definition. The description of how
the circuit is designed is contained within two keywords, CON and END. For our
example nand gate, we would have to put the following instructions inside the
CON and END 'tags' or keywords:
```
CON
    t1 = and(i1, i2)
    o1 = not(t1)
END
```

If you feel like doing it all in one line, you can do that as well ! Just do
it like this:
```
CON
    o1 = not(and(i1, i2))
END
```

Thats it for the HDL ! Let's move on to the Script then !

## Bru scripts
The script files in Bru have different syntax when it comes to combinational
and sequential circuits. We'll start with combinational circuit scripts first.

### Combinational Circuit Scripts
These scripts are very straightforward. All you need to know to write a script
are the names of the inputs given to the circuit marked with SIM, and the 'call'
keyword. So for our nand gate, the script would look something like this:
```
i1 = 1
i2 = 1
call
```

This will cause the bru program to print the result of (1 NAND 1) to the
standard output, your terminal. If you pile up multiple such entries, it will
look something like this:
```
i1 = 1
i2 = 1
call

i1 = 0
i2 = 1
call

i1 = 0
i2 = 0
call

i1 = 1
i2 = 0
call
```

This is effectively going to print out the truth table for our nand gate.

### Sequential Circuit Scripts
For these scripts, there are certain rules that should be kept in mind. One of
these rules is that any sequential circuit script must start by declaring the 
number of cycles to simulate the circuit for. This is done by assigning a whole
number value (integer >= 0) to the 'dur' property in the script, like so:
```
dur = 5
```

Once this is done, you can provide the values for the inputs during different
cycles using the syntax below. It is important to know that if no input is 
specified for a cycle 'n', then the values of the input in cycle 'n-1' are 
carried over to be the values of inputs in cycle 'n'.
```
t = 0 {
    an_input = 1
    another_input = 0
}

(Note: since there are no inputs specified for t = 1, the inputs for t = 1 will
be the same as those for t = 0)

t = 2 {
    an_input = 0
    another_input = 1
}

t = 3 {
    another_input = 1
}
```

(Note: since the value of 'an_input' is not provided at t = 3, the value of 
'an_input' at t = 3 is taken to be what it was at t = 2, which was 0 in this
case)

That's it ! That's all that there is to Bru ! Now its up to you and your
creativity to come up with all kinds of different circuits using this tool.
