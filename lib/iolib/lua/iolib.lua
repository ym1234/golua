local function testf(f)
    return function(...)
        (function(ok, ...) 
            if ok then
                print("OK", ...)
            else
                print("ERR", ...)
            end
        end)(pcall(f, ...))
    end
end

do
    local close = testf(io.close)
    close("abc")
    --> ~ERR	.*must be a file

    print(pcall(io.open))
    --> ~^false\t.*value needed

    print(pcall(io.open, {}))
    --> ~^false\t.*must be a string

    print(pcall(io.open, "aaa", false))
    --> ~^false\t.*must be a string

    testf(io.open)("files/doesnotexist")
    --> ~ERR	.*no such file

    local f = io.open("files/iotest.txt")
    print(f)
    --> =file("files/iotest.txt")

    print(pcall(f.read))
    --> ~^false\t.*value needed

    print(pcall(f.read, 123))
    --> ~^false\t

    print(pcall(f.read, f, "?"))
    --> ~^false\t.*invalid format

    testf(io.type)()
    --> ~ERR	.*value needed

    print(io.type(123))
    --> =nil

    print(io.type(f))
    --> =file

    print(pcall(f.lines))
    --> ~^false\t.*value needed

    print(pcall(f.lines, 123))
    --> ~^false\t.*must be a file

    print(pcall(f.lines, f, "wat"))
    --> ~^false\t.*invalid format


    for line in f:lines() do
        print(line)
    end
    --> =hello
    --> =123
    --> =bye

    testf(f.close)()
    --> ~ERR	.*value needed

    testf(f.flush)()
    --> ~ERR	.*value needed

    testf(f.flush)(123)
    --> ~ERR	.*must be a file

    f:close()
    print(io.type(f))
    --> =closed file
end

do
    testf(io.lines)(123)
    --> ~ERR	.*must be a string

    testf(io.lines)("nonexistent")
    --> ~ERR	.*no such file

    testf(io.lines)("files/iotest.txt", "z")
    --> ~ERR	.*invalid format

    for line in io.lines("files/iotest.txt") do
        print(line)
    end
    --> =hello
    --> =123
    --> =bye

    print((pcall(io.lines, "files/missing")))
    --> =false
end

do
    local function wp(x)
        print("[" .. x .. "]")
    end
    local f = io.open("files/writetest.txt", "w")

    testf(f.write)()
    --> ~ERR	.*value needed

    print(pcall(f.write, 123))
    --> ~^false\t.*must be a file

    print(pcall(f.write, f, {}))
    --> ~^false\t.*must be a string or a number

    f:write("foobar", 1234, "\nabc\n")
    f:close()
    f = io.open("files/writetest.txt", "r")

    wp(f:read("a"))
    --> =[foobar1234
    --> =abc
    --> =]

    wp(f:read("l"))
    --> =[nil]

    testf(f.seek)(f, "set", "hello")
    --> ~ERR	.*must be an integer

    f:seek("set", 0)
    wp(f:read(7))
    --> =[foobar1]

    f:seek("cur", 3)
    wp(f:read(10))
    --> =[
    --> =abc
    --> =]

    f:seek("end", -4)
    wp(f:read("L"))
    --> =[abc
    --> =]

    f:seek("end", -2)
    f:seek("cur", -2)
    wp(f:read("L"))
    --> =[abc
    --> =]
    
    print(pcall(f.seek))
    --> ~false\t.*value needed

    print((pcall(f.seek, "hello")))
    --> =false

    print(pcall(f.seek, f, 42))
    --> ~^false\t.*string

    print(f:seek("set", -100000))
    --> ~^nil\t

    local metaf = getmetatable(f)

    testf(metaf.__tostring)()
    --> ~ERR	.*value needed

    testf(metaf.__tostring)("not a file")
    --> ~ERR	.*must be a file
end

do
    testf(io.read)("z")
    --> ~ERR	.*invalid format

    testf(io.read)(false)
    --> ~ERR	.*invalid format

    local f = io.open("files/writetest2.txt", "w+")
    io.output(f)
    io.write([[Dear sir,
Blah blah,

Yours sincerely.
]])
    io.flush()
    io.input(f)
    f:seek("set", 0)
    print(io.read())
    --> =Dear sir,

    for line in io.lines() do
        print(line)
    end
    --> =Blah blah,
    --> =
    --> =Yours sincerely.
end

do
    local f = io.tmpfile()
    print(io.type(f))
    --> =file

    print(pcall(f.seek, f, "wat"))
    --> =false	#1 must be "cur", "set" or "end"

    -- TODO: do something with the file
end

do
    local stdin = io.input()
    print(io.type(stdin))
    --> =file

    io.input("files/iotest.txt")
    print(io.input())
    --> =file("files/iotest.txt")

    print((pcall(io.input, "files/missing")))
    --> =false

    print((pcall(io.input, 123)))
    --> =false
end

do
    local stdout = io.output()
    print(io.type(stdout))
    --> =file

    testf(io.output)(false)
    --> ~^ERR

    io.output("files/outputtest.txt")
    io.write("hello")
    io.write("bye")
    io.close()
    print(io.open("files/outputtest.txt"):read())
    --> =hellobye
end
