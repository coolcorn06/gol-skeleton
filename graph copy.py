import matplotlib.pyplot as plt

x = ['1', '2', '3', '4', '5']##times
y1 = [5031663459, 5072589875, 5023391584, 5056827208, 5113092750]##runtime
y2 = [3045163416, 3039939875, 3047961750, 3091466792, 3034291375]
y3 = [2393021375, 2369400250, 2340316000, 2495068042, 2348258166]
y4 = [2001461625, 2123262583, 2172323709, 2372598917, 2512846416]
y5 = [2525954750, 2279058041, 2290277459, 2515368042, 2325873708]
y6 = [2480285333, 2262425125, 2282905875, 2043852458, 2141644000]
y7 = [2091324209, 1962487625, 2639254292, 2811546334, 2589152458]
y8 = [2389324958, 1920781500, 2019430292, 2156581250, 2005768791]
y9 = [1923848542, 1978934125, 1937328292, 1939356125, 1939862958]
y10 = [1835916291, 1828551833, 1819233083, 1904982041, 1907617875]
y11 = [1775485791, 1809198042, 1843372333, 1757603250, 1753629083]
y12 = [1752766458, 1750138625, 1729428875, 1743686583, 1788669667]
y13 = [1877075083, 1881236459, 1898388166, 1894078458, 1906456667]
y14 = [1812848292, 1819484958, 1826236541, 1916092333, 1865515500]
y15 = [1830757292, 1790151500, 1786772542, 1784150333, 1782119292]
y16 = [1747699125, 1810762333, 1817312000, 1761696916, 1778968167]

plt.plot(x, y1, marker='o', color='blue', label='512x512x1000x1' )
plt.plot(x, y2, marker='o', color='red', label='512x512x1000x2' )
plt.plot(x, y3, marker='o', color='green', label='512x512x1000x3' )
plt.plot(x, y4, marker='o', color='black', label='512x512x1000x4' )
plt.plot(x, y5, marker='o', color='yellow', label='512x512x1000x5' )
plt.plot(x, y6, marker='o', color='pink', label='512x512x1000x6' )
plt.plot(x, y7, marker='o', color='purple', label='512x512x1000x7' )
plt.plot(x, y8, marker='o', color='orange', label='512x512x1000x8' )
plt.plot(x, y9, marker='o', color='brown', label='512x512x1000x9' )
plt.plot(x, y10, marker='o', color='blue', label='512x512x1000x10' )
plt.plot(x, y11, marker='o', color='red', label='512x512x1000x11' )
plt.plot(x, y12, marker='o', color='green', label='512x512x1000x12' )
plt.plot(x, y13, marker='o', color='black', label='512x512x1000x13' )
plt.plot(x, y14, marker='o', color='yellow', label='512x512x1000x14' )
plt.plot(x, y15, marker='o', color='pink', label='512x512x1000x15' )
plt.plot(x, y16, marker='o', color='purple', label='512x512x1000x16' )


plt.title('runtime comparison', fontsize=12)
plt.xlabel('The times of computation', fontsize=12)
plt.ylabel('Runtime (ns/op)', fontsize=12)
plt.grid(True, linestyle=':', alpha=0.6)
plt.legend()

plt.tight_layout()
plt.show()

plt.plot(1, (sum(y1)/5), marker='o', color='blue', label='512x512x1000x1')
plt.plot(2, (sum(y2)/5), marker='o', color='red', label='512x512x1000x2' )
plt.plot(3, (sum(y3)/5), marker='o', color='green', label='512x512x1000x3' )
plt.plot(4, (sum(y4)/5), marker='o', color='black', label='512x512x1000x4' )
plt.plot(5, (sum(y5)/5), marker='o', color='yellow', label='512x512x1000x5' )
plt.plot(6, (sum(y6)/5), marker='o', color='pink', label='512x512x1000x6' )
plt.plot(7, (sum(y7)/5), marker='o', color='purple', label='512x512x1000x7' )
plt.plot(8, (sum(y8)/5), marker='o', color='orange', label='512x512x1000x8' )
plt.plot(9, (sum(y9)/5), marker='o', color='brown', label='512x512x1000x9' )
plt.plot(10, (sum(y10)/5), marker='o', color='blue', label='512x512x1000x10' )
plt.plot(11, (sum(y11)/5), marker='o', color='red', label='512x512x1000x11' )
plt.plot(12, (sum(y12)/5), marker='o', color='green', label='512x512x1000x12' )
plt.plot(13, (sum(y13)/5), marker='o', color='black', label='512x512x1000x13' )
plt.plot(14, (sum(y14)/5), marker='o', color='yellow', label='512x512x1000x14' )
plt.plot(15, (sum(y15)/5), marker='o', color='pink', label='512x512x1000x15' )
plt.plot(16, (sum(y16)/5), marker='o', color='purple', label='512x512x1000x16' )

plt.title('average runtime comparison', fontsize=12)
plt.xlabel('Computation', fontsize=12)
plt.ylabel('Runtime (s/op)', fontsize=12)
plt.grid(True, linestyle=':', alpha=0.6)
plt.legend()

plt.tight_layout()
plt.show()