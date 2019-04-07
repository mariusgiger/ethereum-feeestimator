# implementation of ethereum gasstation algorithm

The basic strategy is to use statistical modelling to predict confirmation times
at all gas prices from 0-100 gwei at the current state of the txpool and minimum
gas prices accepted in blocks over the last 200 blocks. Then, it selects the
gas price that gives the desired confirmation time assuming standard gas offered
(higher than 1m gas is slower).
