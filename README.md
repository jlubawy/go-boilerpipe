word = space delimited character sequence containing at least one number or letter

# shallow text features

    * average word length
    * average sentence length

# heuristics

    * absolute number of words that start with an uppercase letter
        * ratio of these words to total number of words
        * ratio of full-stops to total number of words
        * number of date/time related tokens
        * number of vertical bars '|'
        * link density (anchor percentage) as the number of tokens within an '<a>' tag divided by total number of tokens in the block

