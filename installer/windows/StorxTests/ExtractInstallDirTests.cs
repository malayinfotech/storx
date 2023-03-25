using System;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Storx;

namespace StorxTests
{
    [TestClass]
    public class ExtractInstallDirTests
    {
        [TestMethod]
        public void NullServiceCmd()
        {
            Assert.IsNull(new CustomActionRunner().ExtractInstallDir(null));
        }

        [TestMethod]
        public void EmptyServiceCmd()
        {
            Assert.IsNull(new CustomActionRunner().ExtractInstallDir(""));
        }

        [TestMethod]
        public void MissingConfigDirFlag()
        {
            Assert.IsNull(new CustomActionRunner().ExtractInstallDir("\"C:\\Program Files\\Storx\\Storage Node\\storagenode.exe\" run"));
        }

        [TestMethod]
        public void ValidServiceCmd()
        {
            Assert.AreEqual("C:\\Program Files\\Storx\\Storage Node\\\\",
                new CustomActionRunner().ExtractInstallDir("\"C:\\Program Files\\Storx\\Storage Node\\storagenode.exe\" run  --config-dir \"C:\\Program Files\\Storx\\Storage Node\\\\\""));
        }
    }
}
