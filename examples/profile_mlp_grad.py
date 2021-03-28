import time
import math
import numpy as np

learning_rate = 0.9

def batch_generate(n, batchsize):
    total = n * batchsize
    return np.random.rand(total)

def avgevry2(indata):
    return (indata[0::2] + indata[1::2]) / 2

def base_name(var):
    """Extracts value passed to name= when creating a variable"""
    return var.name.split('/')[-1].split(':')[0]

def tc_mlp_grad(matrix_dim):
    import client.profile.profile as tc_prof
    import tenncor as tc
    n_in = matrix_dim
    n_out = int(n_in / 2)
    batch_size = 1

    # regular mlp
    brain = tc.api.layer.link([
        tc.api.layer.dense([n_in], [matrix_dim]),
        tc.api.layer.bind(tc.api.sigmoid),
        tc.api.layer.dense([matrix_dim], [n_out]),
        tc.api.layer.bind(tc.api.sigmoid),
    ])

    invar = tc.Variable(np.zeros([batch_size, n_in], dtype=float), 'in')
    # out = brain.connect(invar)
    # expected_out = tc.Variable(np.zeros([batch_size, n_out], dtype=float), 'expected_out')
    # err = tc.api.square(expected_out - out)

    train_input = tc.Variable([batch_size, n_in], label='train_input')
    train_output = tc.Variable([batch_size, n_out], label='train_output')

    invar_batch = batch_generate(n_in, batch_size)
    test_batch = batch_generate(n_in, batch_size)
    test_batch_out = avgevry2(test_batch)

    invar.assign(invar_batch)
    train_input.assign(test_batch)
    train_output.assign(test_batch_out)

    train_err = tc.apply_update([brain],
        lambda err, leaves: tc.api.approx.sgd(err, leaves, learning_rate=learning_rate),
        lambda models: tc.api.error.sqr_diff(train_output, models[0].connect(train_input)))
    tc.optimize("external/com_github_mingkaic_tenncor/cfg/optimizations.json")

    tc_prof.remote_profile('localhost:8069', [train_err])

if __name__ == '__main__':
    tc_mlp_grad(1000)
