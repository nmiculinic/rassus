import pdq

L = 50
S = 0.01
pdq.Init("example 7")

nodes = pdq.CreateNode("Kanal", pdq.CEN, pdq.FCFS)
stream = pdq.CreateOpen("Poruka", L)

pdq.SetVisits("Kanal", "Poruka", 1.0/0.7, S)
# pdq.SetDemand("Kanal", "Poruka", S)
pdq.Solve(pdq.CANON)
pdq.Report()
