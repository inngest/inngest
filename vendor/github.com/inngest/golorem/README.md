Generate lorem ipsum for your project.

=============

Usage
-----
import "lorem"


Ranged generators
-----------------
These will generate a string with a variable number 
of elements specified by a range you provide

    // generate a word with at least min letters and at most max letters.
    Word(min, max int) string  

	// generate a sentence with at least min words and at most max words.
	Sentence(min, max int) string

	// generate a paragraph with at least min sentences and at most max sentences.
	Paragraph(min, max int) string


Convenience functions
---------------------
Generate some commonly occuring tidbits

    Host() string
    Email() string
    Url() string


