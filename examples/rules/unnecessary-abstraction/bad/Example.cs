static class Persistence
{
    public static int Persist(int value) => value * 2;
}
interface IStore
{
    int Save(int value);
}
class Store : IStore
{
    public int Save(int value) => Persistence.Persist(value);
}
