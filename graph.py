import matplotlib.pyplot as plt

# Parallel extension benchmark: 512x512 image, 1000 turns, 1-16 threads, 5 runs each (Apple M2)
x = ['1', '2', '3', '4', '5']  ##times
y1 = [8178287250, 6883505542, 7109256500, 6974578000, 6931344750]  ##runtime (ns/op)
y2 = [4116415458, 4115157250, 4147918750, 4110873417, 4103239416]
y3 = [3207377500, 3250585875, 3203860541, 3209180916, 3217480917]
y4 = [2808578958, 2815519750, 2776494583, 2773238042, 2784810542]
y5 = [2704072875, 2749097458, 2713381458, 2766645250, 2719499459]
y6 = [2528209333, 2551559375, 2527321334, 2603340417, 2556990917]
y7 = [2384675042, 2435008417, 2392467917, 2400404375, 2392670084]
y8 = [2330572917, 2310387042, 2321766667, 2381681542, 2355128500]
y9 = [2588581458, 2585479125, 2593090375, 2591238500, 2621148958]
y10 = [2479349458, 2479899708, 2475458125, 2490944625, 2527775583]
y11 = [2403021042, 2428997041, 2445646250, 2416746250, 2417840000]
y12 = [2333769917, 2336984041, 2376972584, 2337539917, 2316344375]
y13 = [2470928250, 2440112458, 2430888125, 2443349208, 2496375125]
y14 = [2382109834, 2390908250, 2444777375, 2377937083, 2382251333]
y15 = [2376047459, 2345173917, 2346639167, 2350881459, 2333665000]
y16 = [2304560917, 2300684958, 2351433333, 2376997708, 2308433542]

ys = [y1, y2, y3, y4, y5, y6, y7, y8, y9, y10, y11, y12, y13, y14, y15, y16]
colors = ['blue', 'red', 'green', 'black', 'yellow', 'pink', 'purple', 'orange',
          'brown', 'blue', 'red', 'green', 'black', 'yellow', 'pink', 'purple']

for i, y in enumerate(ys):
    plt.plot(x, y, marker='o', color=colors[i], label=f'512x512x1000x{i + 1}')

plt.title('runtime comparison (parallel extension)', fontsize=12)
plt.xlabel('The times of computation', fontsize=12)
plt.ylabel('Runtime (ns/op)', fontsize=12)
plt.grid(True, linestyle=':', alpha=0.6)
plt.legend()

plt.tight_layout()
plt.show()

# Mean runtime for each number of threads
threads = list(range(1, 17))
means = [sum(y) / len(y) for y in ys]

plt.plot(threads, means, marker='o', color='blue')
plt.xticks(threads)
plt.title('mean runtime (parallel extension)', fontsize=12)
plt.xlabel('Number of threads', fontsize=12)
plt.ylabel('Mean runtime (ns/op)', fontsize=12)
plt.grid(True, linestyle=':', alpha=0.6)

plt.tight_layout()
plt.show()
