# should pass
# a = 5
# for i in range(0, 10000000):
#     a += (i * 2) % (i + 5) / 2

# print(a)


# should TLE
a = 5
for i in range(0, 10000000):
    a += (i * 2) % (i + 5) / 2
b = 4

print(a)
print(a / 100000)
